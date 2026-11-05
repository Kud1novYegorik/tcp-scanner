package probe

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

func (f dialFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

var blockUntilDone = dialFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
	<-ctx.Done()
	return nil, ctx.Err()
})

var loopback = netip.MustParseAddr("127.0.0.1")

func TestProbeOpenLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := uint16(ln.Addr().(*net.TCPAddr).Port)

	out := Probe(context.Background(), &net.Dialer{}, loopback, port, 5*time.Second)
	if out.Status != Open || out.Err != nil {
		t.Fatalf("got %+v, want open without error", out)
	}
	if out.Duration < 0 {
		t.Fatalf("negative duration %v", out.Duration)
	}
}

func TestProbeClosedLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint16(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	out := Probe(context.Background(), &net.Dialer{}, loopback, port, 10*time.Second)
	if out.Status != Closed {
		t.Fatalf("got %+v, want closed", out)
	}
	if !errors.Is(out.Err, errnoRefused) {
		t.Fatalf("Err = %v, want to wrap refused errno", out.Err)
	}
}

func TestProbeAddressFormat(t *testing.T) {
	tests := []struct {
		ip   string
		port uint16
		want string
	}{
		{"192.168.1.10", 80, "192.168.1.10:80"},
		{"::1", 443, "[::1]:443"},
		{"fe80::1%eth0", 22, "[fe80::1%eth0]:22"},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			var gotNet, gotAddr string
			d := dialFunc(func(_ context.Context, network, address string) (net.Conn, error) {
				gotNet, gotAddr = network, address
				return nil, errors.New("stop")
			})
			Probe(context.Background(), d, netip.MustParseAddr(tt.ip), tt.port, time.Second)
			if gotNet != "tcp" || gotAddr != tt.want {
				t.Fatalf("dialed %s %q, want tcp %q", gotNet, gotAddr, tt.want)
			}
		})
	}
}

type countingConn struct {
	net.Conn
	closes *atomic.Int32
}

func (c countingConn) Close() error {
	c.closes.Add(1)
	return c.Conn.Close()
}

func TestProbeClosesConnOnSuccess(t *testing.T) {
	var closes atomic.Int32
	client, server := net.Pipe()
	defer server.Close()
	d := dialFunc(func(context.Context, string, string) (net.Conn, error) {
		return countingConn{Conn: client, closes: &closes}, nil
	})

	out := Probe(context.Background(), d, loopback, 80, time.Second)
	if out.Status != Open {
		t.Fatalf("got %+v, want open", out)
	}
	if n := closes.Load(); n != 1 {
		t.Fatalf("Close called %d times, want 1", n)
	}
}

func TestProbeConnectTimeout(t *testing.T) {
	out := Probe(context.Background(), blockUntilDone, loopback, 80, 20*time.Millisecond)
	if out.Status != Timeout {
		t.Fatalf("got %+v, want timeout", out)
	}
	if !errors.Is(out.Err, context.DeadlineExceeded) {
		t.Fatalf("Err = %v, want DeadlineExceeded", out.Err)
	}
}

func TestProbeParentCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	start := time.Now()
	out := Probe(ctx, blockUntilDone, loopback, 80, time.Minute)
	if out.Status != Canceled {
		t.Fatalf("got %+v, want canceled", out)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("probe did not stop promptly after cancel")
	}
}

func TestProbeParentDeadlineIsCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	out := Probe(ctx, blockUntilDone, loopback, 80, time.Minute)
	if out.Status != Canceled {
		t.Fatalf("got %+v, want canceled", out)
	}
}

func TestProbeClassifiesDialError(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name string
		err  error
		want Status
	}{
		{"refused", dialErr(errnoRefused), Closed},
		{"unreachable", dialErr(errnoHostUnreach), Unreachable},
		{"other", boom, Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := dialFunc(func(context.Context, string, string) (net.Conn, error) { return nil, tt.err })
			out := Probe(context.Background(), d, loopback, 80, time.Second)
			if out.Status != tt.want || !errors.Is(out.Err, tt.err) {
				t.Fatalf("got %+v, want status %v with Err %v", out, tt.want, tt.err)
			}
		})
	}
}
