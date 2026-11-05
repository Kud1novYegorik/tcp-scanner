package engine

import (
	"context"
	"iter"
	"sync"
)

func Run[J, R any](ctx context.Context, jobs iter.Seq[J], workers int, fn func(context.Context, J) R) <-chan R {
	workers = max(workers, 1)
	in := make(chan J)
	out := make(chan R)

	var wg sync.WaitGroup
	wg.Add(1 + workers)

	go func() {
		defer wg.Done()
		defer close(in)
		for j := range jobs {
			select {
			case in <- j:
			case <-ctx.Done():
				return
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	for range workers {
		go func() {
			defer wg.Done()
			for j := range in {
				if ctx.Err() != nil {
					return
				}
				r := fn(ctx, j)
				if ctx.Err() != nil {
					return
				}
				select {
				case out <- r:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
