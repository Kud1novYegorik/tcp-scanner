package probe

import (
	"context"
	"net"
	"net/netip"
	"time"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Outcome struct {
	Status   Status
	Duration time.Duration
	Err      error
}

func Probe(ctx context.Context, d Dialer, ip netip.Addr, port uint16, timeout time.Duration) Outcome {
	addr := netip.AddrPortFrom(ip, port).String()

	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	conn, err := d.DialContext(dctx, "tcp", addr)
	elapsed := time.Since(start)

	if err == nil {
		_ = conn.Close()
		return Outcome{Status: Open, Duration: elapsed}
	}
	if ctx.Err() != nil {
		return Outcome{Status: Canceled, Duration: elapsed, Err: err}
	}
	return Outcome{Status: classify(err), Duration: elapsed, Err: err}
}
