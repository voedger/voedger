/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinallyOperatorBasicUsage(t *testing.T) {
	require := require.New(t)
	t.Run("basic", func(t *testing.T) {
		count := 0
		ctx := context.Background()
		p := NewSyncPipeline(ctx, "test pipeline", WireFunc("first", func(ctx context.Context, work interface{}) (err error) {
			return nil
		}), FinallyOperator[int]("finally op", func(i int) error {
			count++
			return nil
		}))

		err := p.SendSync(42)
		require.NoError(err)
		require.Equal(1, count)
	})

	t.Run("error", func(t *testing.T) {
		count := 0
		ctx := context.Background()
		testErr := errors.New("test error")
		p := NewSyncPipeline(ctx, "test pipeline", WireFunc("first", func(ctx context.Context, work interface{}) (err error) {
			return testErr
		}), FinallyOperator[int]("finally op", func(i int) error {
			count++
			return nil
		}))

		err := p.SendSync(42)
		require.ErrorIs(err, testErr)
		require.Equal(1, count)
	})

	t.Run("error in finally", func(t *testing.T) {
		count := 0
		ctx := context.Background()
		testErr := errors.New("test error")
		testErr2 := errors.New("test error 2")
		p := NewSyncPipeline(ctx, "test pipeline", WireFunc("first", func(ctx context.Context, work interface{}) (err error) {
			return testErr
		}), FinallyOperator[int]("finally op", func(i int) error {
			count++
			return testErr2
		}))

		err := p.SendSync(42)
		require.ErrorIs(err, testErr)
		require.Equal(1, count)
	})

	t.Run("error in finally no final error", func(t *testing.T) {
		count := 0
		ctx := context.Background()
		testErr2 := errors.New("test error 2")
		p := NewSyncPipeline(ctx, "test pipeline", WireFunc("first", func(ctx context.Context, work interface{}) (err error) {
			return nil
		}), FinallyOperator[int]("finally op", func(i int) error {
			count++
			return testErr2
		}))

		err := p.SendSync(42)
		require.ErrorIs(err, testErr2)
		require.Equal(1, count)
	})
}
