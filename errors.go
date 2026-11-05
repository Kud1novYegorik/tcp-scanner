package portscan

import (
	"errors"

	"tcp-scanner/internal/portrange"
	"tcp-scanner/internal/target"
)

var (
	ErrInvalidPort   = portrange.ErrInvalidPort
	ErrNoPorts       = errors.New("portscan: no ports")
	ErrInvalidTarget = target.ErrInvalid
	ErrNoTargets     = target.ErrEmpty
	ErrInvalidConfig = errors.New("portscan: invalid config")
)
