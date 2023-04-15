/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
 */

package pipeline

import (
	"context"
	"sync"
)

type forkOperator struct {
	fork     Fork
	branches []ISyncOperator // note: OpFuncQueryState returned by branch.Prepare() will be ignored
}

func (f forkOperator) Close() {
	for _, branch := range f.branches {
		branch.Close()
	}
}

func (f forkOperator) DoSync(ctx context.Context, work interface{}) (err error) {
	forks := make([]interface{}, len(f.branches))
	for i := range f.branches {
		fork, err := f.fork(work, i)
		if err != nil {
			return err
		}
		if fork == nil {
			panic("fork is nil")
		}
		forks[i] = fork
	}

	wg := sync.WaitGroup{}
	errs := make(chan error, len(f.branches))

	for i, branch := range f.branches {
		wg.Add(1)
		go func(i int, branch ISyncOperator) {
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

type ForkOperatorOptionFunc func(*forkOperator)

func ForkOperator(fork Fork, branch ForkOperatorOptionFunc, branches ...ForkOperatorOptionFunc) ISyncOperator {
	if fork == nil {
		panic("fork must be not nil")
	}
	forkOperator := new(forkOperator)
	forkOperator.fork = fork
	branch(forkOperator)
	for _, branch := range branches {
		branch(forkOperator)
	}
	return forkOperator
}

func ForkBranch(o ISyncOperator) ForkOperatorOptionFunc {
	return func(forkOperator *forkOperator) {
		forkOperator.branches = append(forkOperator.branches, o)
	}
}
