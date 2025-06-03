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
	outerCtx context.Context,
	source []IN,
	workers int,
	mapper func(IN) (OUT, error),
	gatherer func(OUT),
) (err error) {
	if workers <= 0 {
		workers = 1
	}

	// errgroup propagates the first non‑nil error and automatically cancels ctx.
	g, workersCtx := errgroup.WithContext(outerCtx)

	// Fan‑out stage – feed the tasks channel
	tasks := make(chan IN)
	g.Go(func() error {
		defer close(tasks)
		for _, v := range source {
			select {
			case <-workersCtx.Done():
				return nil
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
			for workersCtx.Err() == nil {
				select {
				case <-workersCtx.Done():
					return nil
				case in, ok := <-tasks:
					if !ok {
						return nil
					}
					out, err := mapper(in)
					if err != nil {
						return err
					}
					select {
					case results <- out:
					case <-workersCtx.Done():
						return nil
					}
				}
			}
			return nil
		})
	}

	// Close results only after every worker finished.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Gather stage – single goroutine, so caller does not need extra locks
	g.Go(func() error {
		for {
			select {
			case <-workersCtx.Done():
				return nil
			case r, ok := <-results:
				if !ok { // closed by the closer goroutine above
					return nil
				}
				gatherer(r)
			}
		}
	})
	if err = g.Wait(); err == nil {
		err = outerCtx.Err()
	}
	return err
}
