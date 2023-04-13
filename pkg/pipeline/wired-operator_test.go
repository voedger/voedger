/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWiredOperator_forwardIfError(t *testing.T) {
	t.Run("Should be false", func(t *testing.T) {
		operator := WiredOperator{}

		forward := operator.forwardIfErrorAsync(testWorkpiece{})

		require.False(t, forward)
	})
	t.Run("Should be true", func(t *testing.T) {
		operator := WiredOperator{
			Stdout:   make(chan interface{}, 1),
			Operator: &AsyncNOOP{},
		}
		defer close(operator.Stdout)

		forward := operator.forwardIfErrorAsync(errPipeline{})

		require.True(t, forward)
	})
	t.Run("Should panic when workpiece is nil", func(t *testing.T) {
		operator := WiredOperator{
			name: "operator-name",
			wctx: NewWorkpieceContext("pipeline-name", "pipeline-structure"),
		}
		require.PanicsWithValue(t, "critical error in operator 'operator-name': nil in puller_async stdin. "+
			"Pipeline 'pipeline-name' [pipeline-structure]", func() {
			operator.forwardIfErrorAsync(nil)
		})
	})
}

func TestWiredOperator_doAsync(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		operator := WiredOperator{
			Operator: mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					outWork = newTestWork()
					return
				}).
				create(),
		}

		work, err := operator.doAsync(testWorkpiece{})

		require.IsType(t, testwork{}, work)
		require.Nil(t, err)
	})
	t.Run("Should return error on nil work", func(t *testing.T) {
		operator := WiredOperator{
			Operator: mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					err = errors.New("boom")
					return
				}).
				create(),
		}

		work, err := operator.doAsync(testWorkpiece{})

		require.Nil(t, work)
		require.Equal(t, "[/doAsync, outWork==nil] boom", err.Error())
	})
	t.Run("Should return error on not nil work", func(t *testing.T) {
		operator := WiredOperator{
			Operator: mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					outWork = newTestWork()
					err = errors.New("boom")
					return
				}).
				create(),
		}

		work, err := operator.doAsync(testWorkpiece{})

		require.Nil(t, work)
		require.Equal(t, "[/doAsync, outWork!=nil] boom", err.Error())
	})
}
