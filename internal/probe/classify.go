package probe

import (
	"context"
	"errors"
	"net"
)

func classify(err error) Status {
	switch {
	case err == nil:
		return Open
	case errors.Is(err, context.Canceled):
		return Canceled
	case errors.Is(err, errnoRefused):
		return Closed
	case isTimeout(err):
		return Timeout
	case isUnreachable(err):
		return Unreachable
	default:
		return Error
	}
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errnoTimedOut) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func isUnreachable(err error) bool {
	return errors.Is(err, errnoNetUnreach) ||
		errors.Is(err, errnoHostUnreach) ||
		errors.Is(err, errnoNetDown) ||
		errors.Is(err, errnoHostDown)
}
