package portscan

import (
	"net"
	"time"

	"tcp-scanner/internal/probe"
)

type State uint8

const (
	StateUnknown State = iota
	StateOpen
	StateClosed
	StateTimeout
	StateUnreachable
	StateCanceled
	StateError
)

func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateClosed:
		return "closed"
	case StateTimeout:
		return "timeout"
	case StateUnreachable:
		return "unreachable"
	case StateCanceled:
		return "canceled"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

func stateFrom(s probe.Status) State {
	switch s {
	case probe.Open:
		return StateOpen
	case probe.Closed:
		return StateClosed
	case probe.Timeout:
		return StateTimeout
	case probe.Unreachable:
		return StateUnreachable
	case probe.Canceled:
		return StateCanceled
	case probe.Error:
		return StateError
	default:
		return StateUnknown
	}
}

type Result struct {
	Host     string
	IP       net.IP
	Port     uint16
	State    State
	Duration time.Duration
	Err      error
}
