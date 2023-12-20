/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
 */

package pipeline

import (
	"context"
	"reflect"
	"sync"
)

type forkOperator[T any] struct {
	fork     Fork[T]
	branches []ISyncOperator[T] // note: OpFuncQueryState returned by branch.Prepare() will be ignored
}

func (f forkOperator[T]) Close() {
	for _, branch := range f.branches {
		branch.Close()
	}
}

// IsNilPointer checks if a generic type parameter T is a pointer and is nil.
func IsNilPointer[T any](v T) bool {
	// Use reflection to get the type and value of v.
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	// Check if the type of v is a pointer.
	if typ.Kind() == reflect.Ptr {
		// Check if the value of v is nil.
		return val.IsNil()
	}

	// v is not a pointer or is not nil.
	return false
}

func (f forkOperator[T]) DoSync(ctx context.Context, work T) (err error) {
	forks := make([]T, len(f.branches))
	for i := range f.branches {
		fork, err := f.fork(work, i)
		if err != nil {
			return err
		}
		if IsNilPointer(fork) {
			panic("fork is nil")
		}
		forks[i] = fork
	}

	wg := sync.WaitGroup{}
	errs := make(chan error, len(f.branches))

	for i, branch := range f.branches {
		wg.Add(1)
		go func(i int, branch ISyncOperator[T]) {
			defer wg.Done()
			e := branch.DoSync(ctx, forks[i])
			if e != nil {
				errs <- e
			}
		}(i, branch)
	}

	wg.Wait()
	close(errs)

	if len(errs) == 0 {
		return nil
	}

	errInBranches := ErrInBranches{}
	for e := range errs {
		errInBranches.Errors = append(errInBranches.Errors, e)
	}

	return errInBranches
}

type ForkOperatorOptionFunc[T any] func(*forkOperator[T])

func ForkOperator[T any](fork Fork[T], branch ForkOperatorOptionFunc[T], branches ...ForkOperatorOptionFunc[T]) ISyncOperator[T] {
	if fork == nil {
		panic("fork must be not nil")
	}
	forkOperator := new(forkOperator[T])
	forkOperator.fork = fork
	branch(forkOperator)
	for _, branch := range branches {
		branch(forkOperator)
	}
	return forkOperator
}

func ForkBranch[T any](o ISyncOperator[T]) ForkOperatorOptionFunc[T] {
	return func(forkOperator *forkOperator[T]) {
		forkOperator.branches = append(forkOperator.branches, o)
	}
}
