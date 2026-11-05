package target

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/netip"
	"strings"
)

var (
	ErrInvalid   = errors.New("invalid target")
	ErrEmpty     = errors.New("no targets")
	ErrNoAddress = errors.New("no addresses")
)

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type Item struct {
	Host string
	Addr netip.Addr
	Err  error
}

func Validate(hosts []string) error {
	if len(hosts) == 0 {
		return ErrEmpty
	}
	for i, h := range hosts {
		if _, _, err := parse(h); err != nil {
			return fmt.Errorf("hosts[%d]: %w", i, err)
		}
	}
	return nil
}

func Resolve(ctx context.Context, r Resolver, hosts []string, all bool) iter.Seq[Item] {
	return func(yield func(Item) bool) {
		seenIP := make(map[netip.Addr]struct{})
		seenName := make(map[string]struct{})

		emit := func(host string, ip netip.Addr) bool {
			if _, dup := seenIP[ip]; dup {
				return true
			}
			seenIP[ip] = struct{}{}
			return yield(Item{Host: host, Addr: ip})
		}

		for _, raw := range hosts {
			if ctx.Err() != nil {
				return
			}
			host, ip, err := parse(raw)
			if err != nil {
				if !yield(Item{Host: host, Err: err}) {
					return
				}
				continue
			}
			if ip.IsValid() {
				if !emit(host, ip) {
					return
				}
				continue
			}

			key := strings.ToLower(strings.TrimSuffix(host, "."))
			if _, dup := seenName[key]; dup {
				continue
			}
			seenName[key] = struct{}{}

			addrs, err := r.LookupNetIP(ctx, "ip", host)
			if ctx.Err() != nil {
				return
			}
			if err == nil && len(addrs) == 0 {
				err = ErrNoAddress
			}
			if err != nil {
				if !yield(Item{Host: host, Err: fmt.Errorf("resolve %q: %w", host, err)}) {
					return
				}
				continue
			}
			if !all {
				addrs = addrs[:1]
			}
			for _, a := range addrs {
				if !emit(host, a.Unmap()) {
					return
				}
			}
		}
	}
}

func parse(raw string) (host string, ip netip.Addr, err error) {
	host = strings.TrimSpace(raw)
	if host == "" {
		return host, netip.Addr{}, fmt.Errorf("%w: empty host", ErrInvalid)
	}
	if ip, perr := netip.ParseAddr(host); perr == nil {
		return host, ip.Unmap(), nil
	}
	if looksLikeIP(host) {
		return host, netip.Addr{}, fmt.Errorf("%w: %q is not a valid IP address", ErrInvalid, host)
	}
	if !validName(host) {
		return host, netip.Addr{}, fmt.Errorf("%w: %q is not a valid host name", ErrInvalid, host)
	}
	return host, netip.Addr{}, nil
}

func looksLikeIP(s string) bool {
	if strings.Contains(s, ":") {
		return true
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func validName(s string) bool {
	s = strings.TrimSuffix(s, ".")
	if s == "" || len(s) > 253 {
		return false
	}
	for label := range strings.SplitSeq(s, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
				c >= '0' && c <= '9' || c == '-' || c == '_'
			if !ok {
				return false
			}
		}
	}
	return true
}
