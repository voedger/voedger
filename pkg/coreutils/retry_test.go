/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/testingu"
)

func TestRetry(t *testing.T) {
	t.Run("succeed on first attempt", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0
		f := func() error {
			attempts++
			return nil
		}
		err := Retry(ctx, testingu.MockTime, f)
		require.NoError(t, err)
		require.Equal(t, 1, attempts)
	})

	t.Run("retry and succeed after failures", func(t *testing.T) {
		ctx := context.Background()
		tm := testingu.NewMockTime()
		attempts := 0
		f := func() error {
			attempts++
			if attempts < 3 {
				tm.FireNextTimerImmediately()
				return errors.New("temporary error")
			}
			return nil
		}
		err := Retry(ctx, tm, f)
		require.NoError(t, err)
		require.Equal(t, 3, attempts)
	})

	t.Run("context canceled during retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		tm := testingu.NewMockTime()
		attempts := 0
		testErr := errors.New("persistent error")
		f := func() error {
			attempts++
			if attempts == 2 {
				cancel()
			}
			tm.FireNextTimerImmediately()
			return testErr
		}
		err := Retry(ctx, tm, f)
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 2, attempts)
	})
}
