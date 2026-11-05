package portscan

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"
)

const (
	defaultConcurrency    = 100
	defaultConnectTimeout = time.Second
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type config struct {
	concurrency int
	timeout     time.Duration
	resolveAll  bool
	dialer      Dialer
	resolver    Resolver
}

type Option func(*config)

func WithConcurrency(n int) Option {
	return func(c *config) { c.concurrency = n }
}

func WithConnectTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

func WithResolveAll(all bool) Option {
	return func(c *config) { c.resolveAll = all }
}

func WithResolver(r Resolver) Option {
	return func(c *config) { c.resolver = r }
}

func WithDialer(d Dialer) Option {
	return func(c *config) { c.dialer = d }
}

func (c config) validate() error {
	switch {
	case c.concurrency <= 0:
		return fmt.Errorf("%w: concurrency must be positive, got %d", ErrInvalidConfig, c.concurrency)
	case c.timeout <= 0:
		return fmt.Errorf("%w: connect timeout must be positive, got %v", ErrInvalidConfig, c.timeout)
	case c.dialer == nil:
		return fmt.Errorf("%w: dialer is nil", ErrInvalidConfig)
	case c.resolver == nil:
		return fmt.Errorf("%w: resolver is nil", ErrInvalidConfig)
	}
	return nil
}
