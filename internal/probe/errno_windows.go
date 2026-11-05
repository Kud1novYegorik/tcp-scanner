//go:build windows

package probe

import "syscall"

const (
	errnoRefused     = syscall.Errno(10061) // WSAECONNREFUSED
	errnoNetUnreach  = syscall.Errno(10051) // WSAENETUNREACH
	errnoHostUnreach = syscall.Errno(10065) // WSAEHOSTUNREACH
	errnoNetDown     = syscall.Errno(10050) // WSAENETDOWN
	errnoHostDown    = syscall.Errno(10064) // WSAEHOSTDOWN
	errnoTimedOut    = syscall.Errno(10060) // WSAETIMEDOUT
)
