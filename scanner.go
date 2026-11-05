package portscan

import (
	"context"
	"net"
	"net/netip"
	"slices"

	"tcp-scanner/internal/engine"
	"tcp-scanner/internal/probe"
	"tcp-scanner/internal/target"
)

type Scanner struct {
	cfg config
}

func New(opts ...Option) (*Scanner, error) {
	cfg := config{
		concurrency: defaultConcurrency,
		timeout:     defaultConnectTimeout,
		resolveAll:  true,
		dialer:      &net.Dialer{},
		resolver:    net.DefaultResolver,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Scanner{cfg: cfg}, nil
}

type job struct {
	host string
	ip   netip.Addr
	port uint16
	err  error
}

func (s *Scanner) Scan(ctx context.Context, hosts []string, ports PortSet) (<-chan Result, error) {
	if err := target.Validate(hosts); err != nil {
		return nil, err
	}
	if err := ports.validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	hosts = slices.Clone(hosts)
	cfg := s.cfg

	jobs := func(yield func(job) bool) {
		for it := range target.Resolve(ctx, cfg.resolver, hosts, cfg.resolveAll) {
			if it.Err != nil {
				if !yield(job{host: it.Host, err: it.Err}) {
					return
				}
				continue
			}
			for port := range ports.All() {
				if !yield(job{host: it.Host, ip: it.Addr, port: port}) {
					return
				}
			}
		}
	}

	run := func(ctx context.Context, j job) Result {
		if j.err != nil {
			return Result{Host: j.host, State: StateError, Err: j.err}
		}
		out := probe.Probe(ctx, cfg.dialer, j.ip, j.port, cfg.timeout)
		return Result{
			Host:     j.host,
			IP:       net.IP(j.ip.AsSlice()),
			Port:     j.port,
			State:    stateFrom(out.Status),
			Duration: out.Duration,
			Err:      out.Err,
		}
	}

	return engine.Run(ctx, jobs, cfg.concurrency, run), nil
}
