//go:build unix

package probe

import "syscall"

const (
	errnoRefused     = syscall.ECONNREFUSED
	errnoNetUnreach  = syscall.ENETUNREACH
	errnoHostUnreach = syscall.EHOSTUNREACH
	errnoNetDown     = syscall.ENETDOWN
	errnoHostDown    = syscall.EHOSTDOWN
	errnoTimedOut    = syscall.ETIMEDOUT
)
