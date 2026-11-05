package portscan

import (
	"errors"
	"iter"

	"tcp-scanner/internal/portrange"
)

type PortSet struct {
	set portrange.Set
	err error
}

func Port(ports ...int) PortSet {
	parts := make([]PortSet, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, Range(p, p))
	}
	return Ports(parts...)
}

func Range(from, to int) PortSet {
	set, err := portrange.Range(from, to)
	return PortSet{set: set, err: err}
}

func Ports(parts ...PortSet) PortSet {
	sets := make([]portrange.Set, 0, len(parts))
	for _, p := range parts {
		if p.err != nil {
			return PortSet{err: p.err}
		}
		sets = append(sets, p.set)
	}
	return PortSet{set: portrange.Union(sets...)}
}

func ParsePorts(s string) (PortSet, error) {
	set, err := portrange.Parse(s)
	if errors.Is(err, portrange.ErrEmpty) {
		return PortSet{}, ErrNoPorts
	}
	if err != nil {
		return PortSet{}, err
	}
	return PortSet{set: set}, nil
}

func (p PortSet) Err() error { return p.err }

func (p PortSet) Len() int { return p.set.Len() }

func (p PortSet) All() iter.Seq[uint16] { return p.set.All() }

func (p PortSet) validate() error {
	if p.err != nil {
		return p.err
	}
	if p.set.Len() == 0 {
		return ErrNoPorts
	}
	return nil
}
