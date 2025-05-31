
/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ScatterGather implements concurrent mapping the source values and single-goroutine gathering
// it concurrently maps every element from the input slice using the provided
// mapper function and then feeds the produced values to the gatherer.
//
//   - source          – slice with the values to process.
//   - workers         – size of the worker‑pool (≤ 0 defaults to 1).
//   - mapper(IN)      – pure transformation that must be free of side‑effects and may
//     return an error. On the first error every goroutine is
//     cancelled and the error is propagated to the caller.
//   - gatherer(OUT)   – accumulation step that receives the mapped values. Run in **single
//     goroutine**, so it does not have to implement its own synchronisation.
//
// The function returns when every value has been gathered or when any mapper
// returns an error or ctx is cancelled. In that case the first error is returned.
func ScatterGather[IN any, OUT any](
	ctx context.Context,
	source []IN,
	workers int,
	mapper func(IN) (OUT, error),
	gatherer func(OUT),
) error {
	if workers <= 0 {
		workers = 1
	}

	// errgroup propagates the first non‑nil error and automatically cancels ctx.
	g, ctx := errgroup.WithContext(ctx)

	// Fan‑out stage – feed the tasks channel
	tasks := make(chan IN)
	g.Go(func() error {
		defer close(tasks)
		for _, v := range source {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case tasks <- v:
			}
		}
		return nil
	})

	// Map stage – a pool of workers executes the user‑supplied mapper
	results := make(chan OUT)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case in, ok := <-tasks:
					if !ok {
						return nil // channel closed, nothing left to do
					}
					out, err := mapper(in)
					if err != nil {
						return err
					}
					select {
					case results <- out:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		})
	}

	// Close results only after every worker finished.
	g.Go(func() error {
		wg.Wait()
		close(results)
		return nil
	})

	// Gather stage – single goroutine, so caller does not need extra locks
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case r, ok := <-results:
				if !ok { // closed by the closer goroutine above
					return nil
				}
				gatherer(r)
			}
		}
	})

	return g.Wait()
}
