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

// ScetterGather implements parallel source processing using the mappper func and aggregate the result using gatherer func
// mapper produces the OUT instance that then goes to gatherer func
// each element in source is processed by workers pool of size numberOfThreads
// note: not necessary to lock the outer resource to gather into
func ScatterGather[IN any, OUT any](ctx context.Context, source []IN, numberOfThreads int, mapper func(val IN) (OUT, error),
	gatherer func(val OUT)) error {
	g, reducersCtx := errgroup.WithContext(ctx)
	sourceCh := make(chan IN)
	go func() {
		defer close(sourceCh)
		for _, src := range source {
			select {
			case sourceCh <- src:
			case <-ctx.Done():
				return
			}
		}
	}()

	out := make(chan OUT)
	wg := sync.WaitGroup{}
	wg.Add(numberOfThreads)
	for range numberOfThreads {
		g.Go(func() error {
			defer wg.Done()
			return worker(reducersCtx, sourceCh, mapper, out)
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	// reducer
	g.Go(func() error {
		for reducersCtx.Err() == nil {
			select {
			case <-reducersCtx.Done():
				return reducersCtx.Err() // propogate the outer context closing
			case d, ok := <-out:
				if !ok {
					return nil
				}
				gatherer(d)
			}
		}
		return reducersCtx.Err()
	})
	return g.Wait()
}

func worker[IN any, OUT any](ctx context.Context, sourceCh <-chan IN, mapper func(IN) (OUT, error), out chan<- OUT) error {
	for ctx.Err() == nil {
		select {
		case srcVal, ok := <-sourceCh:
			if !ok {
				return nil
			}
			if ctx.Err() != nil {
				// consider valuable error only below
				return nil
			}
			mapped, err := mapper(srcVal)
			if err != nil {
				return err
			}
			select {
			case out <- mapped:
			case <-ctx.Done():
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
	return ctx.Err()
}
