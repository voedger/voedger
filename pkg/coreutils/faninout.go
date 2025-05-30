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

func FanInOut[IN any, OUT any](ctx context.Context, source []IN, numberOfThreads int, sourceHandler func(val IN) (OUT, error), gatherer func(val OUT)) error {
	g, workerCtx := errgroup.WithContext(ctx)
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
			return handler(workerCtx, sourceCh, sourceHandler, out)
		})
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	g.Go(func() error {
		for workerCtx.Err() == nil {
			select {
			case <-workerCtx.Done():
				return workerCtx.Err() // propogate the outer context closing
			case d, ok := <-out:
				if !ok {
					return nil
				}
				gatherer(d)
			}
		}
		return workerCtx.Err()
	})
	return g.Wait()
}

func handler[IN any, OUT any](ctx context.Context, sourceCh <-chan IN, handler func(IN) (OUT, error), out chan<- OUT) error {
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
			d, err := handler(srcVal)
			if err != nil {
				return err
			}
			select {
			case out <- d:
			case <-ctx.Done():
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
	return ctx.Err()
}
