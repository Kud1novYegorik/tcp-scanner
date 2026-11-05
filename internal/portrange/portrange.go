package portrange

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
)

const (
	minPort = 1
	maxPort = 65535
)

var (
	ErrInvalidPort = errors.New("invalid port")
	ErrEmpty       = errors.New("no ports")
)

type interval struct{ lo, hi uint16 }

type Set struct{ iv []interval }

func Single(p int) (Set, error) { return Range(p, p) }

func Range(lo, hi int) (Set, error) {
	if err := checkPort(lo); err != nil {
		return Set{}, err
	}
	if err := checkPort(hi); err != nil {
		return Set{}, err
	}
	if lo > hi {
		return Set{}, fmt.Errorf("%w: reversed range %d-%d", ErrInvalidPort, lo, hi)
	}
	return Set{iv: []interval{{uint16(lo), uint16(hi)}}}, nil
}

func checkPort(p int) error {
	if p < minPort || p > maxPort {
		return fmt.Errorf("%w: %d is out of range %d-%d", ErrInvalidPort, p, minPort, maxPort)
	}
	return nil
}

func Union(sets ...Set) Set {
	var all []interval
	for _, s := range sets {
		all = append(all, s.iv...)
	}
	if len(all) == 0 {
		return Set{}
	}
	slices.SortFunc(all, func(a, b interval) int { return cmp.Compare(a.lo, b.lo) })

	merged := all[:1]
	for _, cur := range all[1:] {
		last := &merged[len(merged)-1]
		if int(cur.lo) <= int(last.hi)+1 {
			last.hi = max(last.hi, cur.hi)
			continue
		}
		merged = append(merged, cur)
	}
	return Set{iv: merged}
}

func Parse(s string) (Set, error) {
	if strings.TrimSpace(s) == "" {
		return Set{}, ErrEmpty
	}
	var parts []Set
	for tok := range strings.SplitSeq(s, ",") {
		tok = strings.TrimSpace(tok)
		loStr, hiStr, isRange := strings.Cut(tok, "-")
		lo, err := strconv.Atoi(strings.TrimSpace(loStr))
		if err != nil {
			return Set{}, fmt.Errorf("%w: %q", ErrInvalidPort, tok)
		}
		hi := lo
		if isRange {
			if hi, err = strconv.Atoi(strings.TrimSpace(hiStr)); err != nil {
				return Set{}, fmt.Errorf("%w: %q", ErrInvalidPort, tok)
			}
		}
		set, err := Range(lo, hi)
		if err != nil {
			return Set{}, err
		}
		parts = append(parts, set)
	}
	return Union(parts...), nil
}

func (s Set) Len() int {
	n := 0
	for _, iv := range s.iv {
		n += int(iv.hi) - int(iv.lo) + 1
	}
	return n
}

func (s Set) All() iter.Seq[uint16] {
	return func(yield func(uint16) bool) {
		for _, iv := range s.iv {
			for p := int(iv.lo); p <= int(iv.hi); p++ {
				if !yield(uint16(p)) {
					return
				}
			}
		}
	}
}
