package target

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strings"
	"testing"
)

type fakeResolver struct {
	answers  map[string][]netip.Addr
	errs     map[string]error
	calls    []string
	onLookup func(host string)
}

func (f *fakeResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	f.calls = append(f.calls, host)
	if network != "ip" {
		return nil, errors.New("unexpected network " + network)
	}
	if f.onLookup != nil {
		f.onLookup(host)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := strings.ToLower(host)
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	return f.answers[key], nil
}

func addrs(ss ...string) []netip.Addr {
	out := make([]netip.Addr, 0, len(ss))
	for _, s := range ss {
		out = append(out, netip.MustParseAddr(s))
	}
	return out
}

func resolve(t *testing.T, r Resolver, hosts []string, all bool) []Item {
	t.Helper()
	return slices.Collect(Resolve(context.Background(), r, hosts, all))
}

func TestParseAndValidate(t *testing.T) {
	valid := []string{
		"example.com", "example.com.", "EXAMPLE.com", "localhost", "my_host.local",
		"a-b.c1.io", "192.168.1.10", "::1", "2001:db8::1", "fe80::1%eth0",
		"  10.0.0.1  ", "::ffff:10.0.0.1",
	}
	invalid := []string{
		"", "   ", "\t", "999.1.1.1", "1.2.3", "1.2.3.4.5", "::gg", "[::1]",
		"example.com:80", "http://example.com", "host name", "-bad.com",
		"bad-.com", "a..b", ".", "ünicode.com", "a/b", "1.2.3.4:80",
		strings.Repeat("a", 64) + ".com",
		strings.Repeat("a.", 130) + "com",
	}
	for _, h := range valid {
		if err := Validate([]string{h}); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", h, err)
		}
	}
	for _, h := range invalid {
		if err := Validate([]string{h}); !errors.Is(err, ErrInvalid) {
			t.Errorf("Validate(%q) = %v, want ErrInvalid", h, err)
		}
	}
	if err := Validate(nil); !errors.Is(err, ErrEmpty) {
		t.Errorf("Validate(nil) = %v, want ErrEmpty", err)
	}
	// Одна плохая цель среди хороших роняет валидацию и называет индекс.
	err := Validate([]string{"example.com", "999.1.1.1"})
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "hosts[1]") {
		t.Errorf("Validate mixed = %v", err)
	}
}

func TestResolveLiterals(t *testing.T) {
	r := &fakeResolver{}
	got := resolve(t, r, []string{"192.168.1.10", " ::1 ", "fe80::1%eth0"}, true)
	want := []Item{
		{Host: "192.168.1.10", Addr: netip.MustParseAddr("192.168.1.10")},
		{Host: "::1", Addr: netip.MustParseAddr("::1")},
		{Host: "fe80::1%eth0", Addr: netip.MustParseAddr("fe80::1%eth0")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if len(r.calls) != 0 {
		t.Fatalf("literals must not hit DNS, calls = %v", r.calls)
	}
}

func TestResolveDedupByIP(t *testing.T) {
	r := &fakeResolver{answers: map[string][]netip.Addr{"a.test": addrs("10.0.0.1", "10.0.0.2")}}
	hosts := []string{"10.0.0.1", " 10.0.0.1", "::ffff:10.0.0.1", "a.test"}
	got := resolve(t, r, hosts, true)
	want := []Item{
		{Host: "10.0.0.1", Addr: netip.MustParseAddr("10.0.0.1")}, // первая цель побеждает
		{Host: "a.test", Addr: netip.MustParseAddr("10.0.0.2")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestResolveDedupByName(t *testing.T) {
	r := &fakeResolver{answers: map[string][]netip.Addr{"a.test": addrs("10.0.0.1")}}
	got := resolve(t, r, []string{"A.test", "a.test", "a.test."}, true)
	if len(got) != 1 {
		t.Fatalf("got %+v, want one item", got)
	}
	if len(r.calls) != 1 {
		t.Fatalf("lookups = %v, want exactly one", r.calls)
	}
}

func TestResolveAllVsFirst(t *testing.T) {
	r := &fakeResolver{answers: map[string][]netip.Addr{"multi.test": addrs("10.0.0.1", "2001:db8::1", "10.0.0.2")}}

	all := resolve(t, r, []string{"multi.test"}, true)
	if len(all) != 3 {
		t.Fatalf("all: got %+v, want 3 items", all)
	}
	first := resolve(t, r, []string{"multi.test"}, false)
	if len(first) != 1 || first[0].Addr != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("first: got %+v, want only 10.0.0.1", first)
	}
}

func TestResolveErrorIsPerTarget(t *testing.T) {
	r := &fakeResolver{
		answers: map[string][]netip.Addr{"empty.test": nil},
		errs:    map[string]error{"bad.test": &net.DNSError{Err: "no such host", Name: "bad.test", IsNotFound: true}},
	}
	got := resolve(t, r, []string{"bad.test", "10.0.0.1", "empty.test"}, true)
	if len(got) != 3 {
		t.Fatalf("got %+v, want 3 items", got)
	}

	var dnsErr *net.DNSError
	if got[0].Host != "bad.test" || !errors.As(got[0].Err, &dnsErr) || !dnsErr.IsNotFound {
		t.Fatalf("item 0 = %+v, want wrapped *net.DNSError", got[0])
	}
	if got[1].Err != nil || got[1].Addr != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("item 1 = %+v, want 10.0.0.1 without error", got[1])
	}
	if !errors.Is(got[2].Err, ErrNoAddress) {
		t.Fatalf("item 2 = %+v, want ErrNoAddress", got[2])
	}
}

func TestResolveInvalidBecomesItem(t *testing.T) {
	got := resolve(t, &fakeResolver{}, []string{"999.1.1.1", "10.0.0.1"}, true)
	if len(got) != 2 || !errors.Is(got[0].Err, ErrInvalid) || got[1].Err != nil {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveIsLazy(t *testing.T) {
	r := &fakeResolver{answers: map[string][]netip.Addr{
		"a.test": addrs("10.0.0.1"), "b.test": addrs("10.0.0.2"), "c.test": addrs("10.0.0.3"),
	}}
	for range Resolve(context.Background(), r, []string{"a.test", "b.test", "c.test"}, true) {
		break
	}
	if !slices.Equal(r.calls, []string{"a.test"}) {
		t.Fatalf("lookups = %v, want only a.test", r.calls)
	}
}

func TestResolveCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &fakeResolver{}
	if got := slices.Collect(Resolve(ctx, r, []string{"10.0.0.1", "a.test"}, true)); len(got) != 0 {
		t.Fatalf("got %+v, want nothing", got)
	}
	if len(r.calls) != 0 {
		t.Fatalf("lookups = %v, want none", r.calls)
	}
}

func TestResolveCanceledDuringLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &fakeResolver{
		answers:  map[string][]netip.Addr{"a.test": addrs("10.0.0.1"), "b.test": addrs("10.0.0.2")},
		onLookup: func(string) { cancel() },
	}
	got := slices.Collect(Resolve(ctx, r, []string{"a.test", "b.test", "10.0.0.9"}, true))
	if len(got) != 0 {
		t.Fatalf("got %+v, want nothing after cancel", got)
	}
	if !slices.Equal(r.calls, []string{"a.test"}) {
		t.Fatalf("lookups = %v, want only a.test", r.calls)
	}
}
