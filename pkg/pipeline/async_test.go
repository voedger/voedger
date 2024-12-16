// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_p_release(t *testing.T) {
	t.Run("Should release workpiece", func(t *testing.T) {
		release := false

		p_release(testWorkpiece{func() {
			release = true
		}})

		require.True(t, release)
	})
	t.Run("Should not release workpiece because workpiece is nil", func(t *testing.T) {
		release := false

		p_release(nil)

		require.False(t, release)
	})
}

func Test_p_flush(t *testing.T) {
	testErr := errors.New("error occurred")
	t.Run("Should flush", func(t *testing.T) {
		operator := WireAsyncOperator("operator", mockAsyncOp().
			flush(func(callback OpFuncFlush) (err error) {
				callback(testWorkpiece{})
				return nil
			}).create())
		operator.ctx = context.Background()

		p_flush(operator, "test")

		require.Len(t, operator.Stdout, 1)
		require.Equal(t, testWorkpiece{}, <-operator.Stdout)
	})
	t.Run("Should no flush by error reason", func(t *testing.T) {
		operator := WireAsyncOperator("operator", mockAsyncOp().
			flush(func(callback OpFuncFlush) (err error) {
				return testErr
			}).create())
		operator.ctx = context.Background()

		p_flush(operator, "test")

		require.Len(t, operator.Stdout, 1)
		err := <-operator.Stdout
		require.ErrorIs(t, err.(*errPipeline), testErr)
	})
	t.Run("Should no flush by error in ctx reason", func(t *testing.T) {
		operator := WireAsyncOperator("operator", mockAsyncOp().
			flush(func(callback OpFuncFlush) (err error) {
				callback(testWorkpiece{})
				return nil
			}).create())
		operator.ctx = testContext{err: testErr}

		p_flush(operator, "test")

		require.Empty(t, operator.Stdout)
	})
}

func Test_puller_async(t *testing.T) {
	t.Run("Should release workpiece and continue when wired operator isn not active", func(t *testing.T) {
		release := false
		doAsync := false
		wg := new(sync.WaitGroup)
		wg.Add(1)
		work := testWorkpiece{func() {
			release = true
			wg.Done()
		}}
		operator := &WiredOperator{
			Operator: mockAsyncOp().doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
				doAsync = true
				return nil, err
			}).create(),
			Stdin:  make(chan interface{}, 1),
			Stdout: make(chan interface{}, 1),
			ctx:    context.Background(),
			err:    errPipeline{},
		}
		go puller_async(operator)
		operator.Stdin <- work
		wg.Wait()
		close(operator.Stdout)

		require.True(t, release)
		require.False(t, doAsync)
	})
	t.Run("Should forward error and continue", func(t *testing.T) {
		doAsync := false
		operator := &WiredOperator{
			Operator: mockAsyncOp().doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
				doAsync = true
				return nil, err
			}).create(),
			Stdin:  make(chan interface{}, 1),
			Stdout: make(chan interface{}, 1),
			ctx:    context.Background(),
		}
		go puller_async(operator)
		operator.Stdin <- errPipeline{}
		err := <-operator.Stdout
		close(operator.Stdout)

		require.Equal(t, errPipeline{}, err)
		require.False(t, doAsync)
	})
	t.Run("Should send error when error occurred at doAsync method", func(t *testing.T) {
		doAsync := false
		operator := &WiredOperator{
			Operator: mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					doAsync = true
					return nil, errors.New("boom")
				}).create(),
			Stdin:  make(chan interface{}, 1),
			Stdout: make(chan interface{}, 1),
			ctx:    context.Background(),
		}
		go puller_async(operator)
		operator.Stdin <- testWorkpiece{}
		err := <-operator.Stdout
		close(operator.Stdout)

		require.IsType(t, new(errPipeline), err)
		require.True(t, doAsync)
	})
}
