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

func TestAsyncPipeline_SendAsync(t *testing.T) {
	t.Run("Should return error on ctx termination", func(t *testing.T) {
		expectedErr := errors.New("context termination")
		pipeline := NewAsyncPipeline(
			testContext{err: expectedErr},
			"async-pipeline",
			WireAsyncFunc[IWorkpiece]("async-operator-nop", nil))

		err := pipeline.SendAsync(nil)

		require.Equal(t, expectedErr, err)
	})
	t.Run("Should return error from operator", func(t *testing.T) {
		asyncOperatorError := errors.New("async operator error")
		pipeline := NewAsyncPipeline(
			context.Background(),
			"async-pipeline",
			WireAsyncFunc("async-operator", func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
				return nil, asyncOperatorError
			}),
		)

		var err error
		for err == nil {
			err = pipeline.SendAsync(newTestWork())
		}

		require.ErrorIs(t, err, asyncOperatorError)
	})
	t.Run("Should release workpiece in the end", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(1)
		pipeline := NewAsyncPipeline(
			context.Background(),
			"async-pipeline",
			WireAsyncOperator(
				"async-operator",
				NewAsyncOp(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					return work, nil
				})),
		)

		require.NoError(t, pipeline.SendAsync(testWorkpiece{
			func() {
				wg.Done()
			},
		}))
		wg.Wait()
	})
}

func TestAsyncPipeline_Close(t *testing.T) {
	pipeline := &AsyncPipeline{
		stdin:  make(chan interface{}, 1),
		stdout: make(chan interface{}, 1),
	}
	pipeline.stdout <- newTestWork()
	close(pipeline.stdout)

	require.NotPanics(t, func() {
		pipeline.Close()
	})
}

func TestAsyncPipeline_OnError(t *testing.T) {
	handledErrs := make(chan error)
	pipeline := NewAsyncPipeline(
		context.Background(),
		"async-pipeline",
		WireAsyncOperator(
			"operator1", mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					return nil, errors.New("test error")
				}).
				onError(func(ctx context.Context, err error) {
					panic("Must not be called")
				}).create()),
		WireAsyncOperator(
			"operator2", mockAsyncOp().
				onError(func(ctx context.Context, err error) {
					handledErrs <- err
				}).create()),
		WireAsyncOperator(
			"operator3", mockAsyncOp().
				onError(func(ctx context.Context, err error) {
					handledErrs <- err
				}).create()),
	)
	defer pipeline.Close()

	require.NoError(t, pipeline.SendAsync(newTestWork()))
	var errInPipeline IErrorPipeline
	require.ErrorAs(t, <-handledErrs, &errInPipeline)
	require.Equal(t, "test error", errInPipeline.Error())
	require.Equal(t, "doAsync, outWork==nil", errInPipeline.GetPlace())
	require.Equal(t, "operator1", errInPipeline.GetOpName())

	//var errInPipeline IErrorPipeline
	require.ErrorAs(t, <-handledErrs, &errInPipeline)
	require.Equal(t, "test error", errInPipeline.Error())
	require.Equal(t, "doAsync, outWork==nil", errInPipeline.GetPlace())
	require.Equal(t, "operator1", errInPipeline.GetOpName())
}
