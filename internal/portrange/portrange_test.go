package portrange

import (
	"errors"
	"slices"
	"testing"
)

func collect(s Set) []uint16 { return slices.Collect(s.All()) }

func TestRangeValidation(t *testing.T) {
	tests := []struct {
		name    string
		lo, hi  int
		wantErr bool
		wantLen int
	}{
		{"single min", 1, 1, false, 1},
		{"single max", 65535, 65535, false, 1},
		{"full range", 1, 65535, false, 65535},
		{"zero", 0, 0, true, 0},
		{"zero start", 0, 10, true, 0},
		{"above max", 65536, 65536, true, 0},
		{"above max end", 1, 70000, true, 0},
		{"negative", -1, 5, true, 0},
		{"reversed", 1000, 1, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Range(tt.lo, tt.hi)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidPort) {
					t.Fatalf("err = %v, want ErrInvalidPort", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s.Len() != tt.wantLen {
				t.Fatalf("Len = %d, want %d", s.Len(), tt.wantLen)
			}
		})
	}
}

func TestUnion(t *testing.T) {
	mk := func(lo, hi int) Set {
		s, err := Range(lo, hi)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	tests := []struct {
		name string
		in   []Set
		want []uint16
	}{
		{"empty", nil, nil},
		{"duplicates", []Set{mk(80, 80), mk(80, 80)}, []uint16{80}},
		{"overlap", []Set{mk(1, 5), mk(3, 8)}, []uint16{1, 2, 3, 4, 5, 6, 7, 8}},
		{"adjacent", []Set{mk(1, 3), mk(4, 6)}, []uint16{1, 2, 3, 4, 5, 6}},
		{"contained", []Set{mk(1, 10), mk(3, 4)}, []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"unsorted gap", []Set{mk(10, 11), mk(1, 2)}, []uint16{1, 2, 10, 11}},
		{"max boundary", []Set{mk(65534, 65535), mk(65535, 65535)}, []uint16{65534, 65535}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Union(tt.in...)
			if !slices.Equal(collect(got), tt.want) {
				t.Fatalf("got %v, want %v", collect(got), tt.want)
			}
			if got.Len() != len(tt.want) {
				t.Fatalf("Len = %d, want %d", got.Len(), len(tt.want))
			}
		})
	}
}

func TestUnionMergesIntervals(t *testing.T) {
	a, _ := Range(1, 1000)
	b, _ := Range(500, 2000)
	c, _ := Range(2001, 3000)
	if n := len(Union(a, b, c).iv); n != 1 {
		t.Fatalf("intervals = %d, want 1", n)
	}
}

func TestAllEarlyStop(t *testing.T) {
	s, _ := Range(1, 65535)
	n := 0
	for range s.All() {
		if n++; n == 3 {
			break
		}
	}
	if n != 3 {
		t.Fatalf("n = %d, want 3", n)
	}
}

func TestAllMaxPortTerminates(t *testing.T) {
	s, _ := Range(65535, 65535)
	if got := collect(s); !slices.Equal(got, []uint16{65535}) {
		t.Fatalf("got %v", got)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    []uint16
		wantErr error
	}{
		{"80", []uint16{80}, nil},
		{"22,80,8000-8002", []uint16{22, 80, 8000, 8001, 8002}, nil},
		{" 22 , 80 - 82 ", []uint16{22, 80, 81, 82}, nil},
		{"80,80,79-81", []uint16{79, 80, 81}, nil},
		{"", nil, ErrEmpty},
		{"   ", nil, ErrEmpty},
		{"0", nil, ErrInvalidPort},
		{"65536", nil, ErrInvalidPort},
		{"1000-1", nil, ErrInvalidPort},
		{"22,,80", nil, ErrInvalidPort},
		{"abc", nil, ErrInvalidPort},
		{"1-", nil, ErrInvalidPort},
		{"-5", nil, ErrInvalidPort},
		{"1-2-3", nil, ErrInvalidPort},
		{"99999999999999999999", nil, ErrInvalidPort},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && !slices.Equal(collect(got), tt.want) {
				t.Fatalf("got %v, want %v", collect(got), tt.want)
			}
		})
	}
}

func TestZeroValueIsEmpty(t *testing.T) {
	var s Set
	if s.Len() != 0 || len(collect(s)) != 0 {
		t.Fatal("zero Set must be empty")
	}
}
