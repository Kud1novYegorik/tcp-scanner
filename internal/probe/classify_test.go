package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func dialErr(inner error) error {
	return &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", inner)}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Status
	}{
		{"nil is open", nil, Open},
		{"canceled", context.Canceled, Canceled},
		{"wrapped canceled", fmt.Errorf("scan: %w", context.Canceled), Canceled},
		{"refused", dialErr(errnoRefused), Closed},
		{"bare refused errno", errnoRefused, Closed},
		{"net unreachable", dialErr(errnoNetUnreach), Unreachable},
		{"host unreachable", dialErr(errnoHostUnreach), Unreachable},
		{"net down", dialErr(errnoNetDown), Unreachable},
		{"host down", dialErr(errnoHostDown), Unreachable},
		{"deadline exceeded", context.DeadlineExceeded, Timeout},
		{"wrapped deadline", fmt.Errorf("dial: %w", context.DeadlineExceeded), Timeout},
		{"net.Error timeout", &net.OpError{Op: "dial", Err: timeoutErr{}}, Timeout},
		{"os deadline", &net.OpError{Op: "dial", Err: os.ErrDeadlineExceeded}, Timeout},
		{"os-level timed out", dialErr(errnoTimedOut), Timeout},
		{"other error", errors.New("boom"), Error},
		{"text must not matter", errors.New("connection refused"), Error},
		{"text must not matter 2", errors.New("i/o timeout"), Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classify(tt.err); got != tt.want {
				t.Fatalf("classify(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestStatusString(t *testing.T) {
	want := map[Status]string{
		Unknown: "unknown", Open: "open", Closed: "closed", Timeout: "timeout",
		Unreachable: "unreachable", Canceled: "canceled", Error: "error",
	}
	for s, str := range want {
		if s.String() != str {
			t.Errorf("Status(%d).String() = %q, want %q", s, s.String(), str)
		}
	}
}
