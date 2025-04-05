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
)

func TestRetry_(t *testing.T) {
	t.Run("succeed on first attempt", func(t *testing.T) {
		// Setup test
		ctx := context.Background()
		var attempts int = 0
		f := func() error {
			attempts++
			return nil
		}

		// Call the function under test
		err := Retry(ctx, MockTime, f)

		// Verify results
		require.NoError(t, err)
		require.Equal(t, 1, attempts)
	})

	t.Run("retry and succeed after failures", func(t *testing.T) {
		// Setup test
		ctx := context.Background()
		tm := NewMockTime()
		var attempts int = 0
		f := func() error {
			attempts++
			if attempts < 3 {
				tm.FireNextTimerImmediately()
				return errors.New("temporary error")
			}
			return nil
		}

		// Call the function under test
		err := Retry(ctx, tm, f)

		// Verify results
		require.NoError(t, err)
		require.Equal(t, 3, attempts)
	})

	t.Run("context canceled during retry", func(t *testing.T) {
		// Setup test
		ctx, cancel := context.WithCancel(context.Background())
		tm := NewMockTime()
		var attempts int = 0
		testErr := errors.New("persistent error")
		f := func() error {
			attempts++
			if attempts == 2 {
				cancel()
			}
			tm.FireNextTimerImmediately()
			return testErr
		}

		// Call the function under test
		err := Retry(ctx, tm, f)

		// Verify results
		require.ErrorIs(t, err, testErr)
		require.Equal(t, 2, attempts)
	})
}
