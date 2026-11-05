package portscan

import (
	"errors"
	"slices"
	"testing"
)

func TestPortSetConstructors(t *testing.T) {
	tests := []struct {
		name    string
		set     PortSet
		want    []uint16
		wantErr error
	}{
		{"single ports", Port(443, 80, 80), []uint16{80, 443}, nil},
		{"range", Range(1, 3), []uint16{1, 2, 3}, nil},
		{"combined", Ports(Port(22), Range(80, 82), Range(81, 84)), []uint16{22, 80, 81, 82, 83, 84}, nil},
		{"zero port", Port(0), nil, ErrInvalidPort},
		{"above max", Range(1, 65536), nil, ErrInvalidPort},
		{"reversed", Range(1000, 1), nil, ErrInvalidPort},
		{"error propagates", Ports(Port(22), Range(1000, 1)), nil, ErrInvalidPort},
		{"no ports", Port(), nil, ErrNoPorts},
		{"zero value", PortSet{}, nil, ErrNoPorts},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.set.validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("validate() = %v, want %v", err, tt.wantErr)
			}
			if got := slices.Collect(tt.set.All()); !slices.Equal(got, tt.want) {
				t.Fatalf("All() = %v, want %v", got, tt.want)
			}
			if tt.set.Len() != len(tt.want) {
				t.Fatalf("Len() = %d, want %d", tt.set.Len(), len(tt.want))
			}
		})
	}
}

func TestPortSetErrIsFirstError(t *testing.T) {
	if err := Range(1000, 1).Err(); !errors.Is(err, ErrInvalidPort) {
		t.Fatalf("Err() = %v", err)
	}
	if err := Range(1, 10).Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
}

func TestParsePorts(t *testing.T) {
	set, err := ParsePorts("22,80,8000-8002")
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{22, 80, 8000, 8001, 8002}
	if got := slices.Collect(set.All()); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if _, err := ParsePorts(""); !errors.Is(err, ErrNoPorts) {
		t.Fatalf("empty: err = %v, want ErrNoPorts", err)
	}
	for _, bad := range []string{"0", "65536", "1000-1", "x", "22,,80"} {
		if _, err := ParsePorts(bad); !errors.Is(err, ErrInvalidPort) {
			t.Fatalf("%q: err = %v, want ErrInvalidPort", bad, err)
		}
	}
}
