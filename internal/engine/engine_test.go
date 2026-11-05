package engine

import (
	"context"
	"iter"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const wait = 10 * time.Second

func seq(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func counting(produced *atomic.Int64) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; ; i++ {
			produced.Add(1)
			if !yield(i) {
				return
			}
		}
	}
}

func drain[R any](t *testing.T, ch <-chan R) []R {
	t.Helper()
	var got []R
	timeout := time.After(wait)
	for {
		select {
		case r, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, r)
		case <-timeout:
			t.Errorf("channel was not closed within %v", wait)
			return got
		}
	}
}

func recv(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(wait):
		t.Fatal("timed out waiting for signal")
	}
}

func requireNoLeak(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(wait)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Fatalf("goroutine leak: %d > %d\n%s", runtime.NumGoroutine(), baseline, buf[:n])
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func double(_ context.Context, j int) int { return j * 2 }

func TestRunProcessesAll(t *testing.T) {
	base := runtime.NumGoroutine()
	got := drain(t, Run(context.Background(), seq(1000), 8, double))
	slices.Sort(got)

	want := make([]int, 1000)
	for i := range want {
		want[i] = i * 2
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %d results, first mismatch somewhere; len want %d", len(got), len(want))
	}
	requireNoLeak(t, base)
}

func TestRunEmptyInput(t *testing.T) {
	base := runtime.NumGoroutine()
	if got := drain(t, Run(context.Background(), seq(0), 4, double)); len(got) != 0 {
		t.Fatalf("got %v, want nothing", got)
	}
	requireNoLeak(t, base)
}

func TestRunWorkersBelowOneActsAsOne(t *testing.T) {
	for _, w := range []int{0, -3} {
		if got := drain(t, Run(context.Background(), seq(10), w, double)); len(got) != 10 {
			t.Fatalf("workers=%d: got %d results, want 10", w, len(got))
		}
	}
}

func TestRunConcurrencyLimit(t *testing.T) {
	const workers = 5
	var cur, peak atomic.Int32
	reached := make(chan struct{})
	var once sync.Once

	fn := func(_ context.Context, j int) int {
		n := cur.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		if n == workers {
			once.Do(func() { close(reached) })
		}
		select {
		case <-reached:
		case <-time.After(wait):
		}
		cur.Add(-1)
		return j
	}

	if got := drain(t, Run(context.Background(), seq(100), workers, fn)); len(got) != 100 {
		t.Fatalf("got %d results, want 100", len(got))
	}
	if p := peak.Load(); p != workers {
		t.Fatalf("peak concurrency = %d, want %d", p, workers)
	}
}

func TestRunCancelStopsWorkAndCloses(t *testing.T) {
	const workers = 4
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var produced, calls atomic.Int64
	started := make(chan struct{}, workers)
	fn := func(ctx context.Context, j int) int {
		calls.Add(1)
		started <- struct{}{}
		<-ctx.Done()
		return j
	}

	out := Run(ctx, counting(&produced), workers, fn)
	for range workers {
		recv(t, started)
	}
	cancel()

	if got := drain(t, out); len(got) > workers {
		t.Fatalf("got %d results after cancel, want at most %d", len(got), workers)
	}
	if p := produced.Load(); p > workers+1 {
		t.Fatalf("produced %d jobs, want at most %d", p, workers+1)
	}
	if c := calls.Load(); c > workers {
		t.Fatalf("fn called %d times, want at most %d", c, workers)
	}

	p := produced.Load()
	time.Sleep(50 * time.Millisecond)
	if produced.Load() != p {
		t.Fatal("jobs were produced after the channel closed")
	}
	requireNoLeak(t, base)
}

func TestRunCanceledBeforeStart(t *testing.T) {
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var produced atomic.Int64
	drain(t, Run(ctx, counting(&produced), 4, double))
	if p := produced.Load(); p > 1 {
		t.Fatalf("produced %d jobs on a canceled context, want at most 1", p)
	}
	requireNoLeak(t, base)
}

func TestRunSlowReaderBackpressure(t *testing.T) {
	const workers = 3
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var produced atomic.Int64
	out := Run(ctx, counting(&produced), workers, double)

	time.Sleep(100 * time.Millisecond)
	if p := produced.Load(); p > workers+1 {
		t.Fatalf("produced %d jobs with no reader, want at most %d", p, workers+1)
	}

	const read = 10
	for range read {
		select {
		case <-out:
		case <-time.After(wait):
			t.Fatal("timed out reading results")
		}
	}
	time.Sleep(50 * time.Millisecond)
	if p := produced.Load(); p > read+workers+1 {
		t.Fatalf("produced %d jobs after reading %d, want at most %d", p, read, read+workers+1)
	}

	cancel()
	drain(t, out)
	requireNoLeak(t, base)
}

func TestRunAbandonedReaderCancel(t *testing.T) {
	base := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())

	var produced atomic.Int64
	out := Run(ctx, counting(&produced), 3, double)
	_ = out

	time.Sleep(50 * time.Millisecond)
	cancel()
	requireNoLeak(t, base)
}

func TestRunConcurrentRuns(t *testing.T) {
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := drain(t, Run(context.Background(), seq(500), 4, double)); len(got) != 500 {
				t.Errorf("got %d results, want 500", len(got))
			}
		}()
	}
	wg.Wait()
}
