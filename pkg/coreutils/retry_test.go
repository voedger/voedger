/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package coreutils

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	t.Run("should succeed on the first attempt", func(t *testing.T) {
		ctx := context.Background()
		iTime := MockTime
		iTime.Add(-iTime.Now().Sub(time.Now())) // reset to Now
		retryDelay := time.Millisecond
		retryCount := 3
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			return nil
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.NoError(t, err)
		require.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	})

	t.Run("should retry and succeed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		iTime := MockTime
		retryDelay := 5 * time.Millisecond
		retryCount := 3
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			iTime.Sleep(retryDelay)
			if atomic.LoadInt32(&attempts) < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.NoError(t, err)
		require.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	})

	t.Run("should retry and fail after retryCount attempts", func(t *testing.T) {
		ctx := context.Background()
		iTime := MockTime
		retryDelay := 5 * time.Millisecond
		retryCount := 3
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			iTime.Sleep(retryDelay)
			return errors.New("persistent error")
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.ErrorIs(t, err, ErrRetryAttemptsExceeded)
		require.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	})

	t.Run("should stop retrying when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		iTime := MockTime
		retryDelay := time.Millisecond
		retryCount := 5
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			if atomic.LoadInt32(&attempts) == 2 {
				cancel()
				time.Sleep(retryDelay)
			}
			iTime.Sleep(retryDelay + time.Millisecond)
			return errors.New("temporary error")
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.Error(t, err)
	})

	t.Run("should retry indefinitely until success with retryCount = 0", func(t *testing.T) {
		ctx := context.Background()
		iTime := MockTime
		retryDelay := time.Millisecond
		retryCount := 0
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			if atomic.LoadInt32(&attempts) < 3 {
				iTime.Sleep(retryDelay)
				return errors.New("temporary error")
			}
			return nil
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.NoError(t, err)
		require.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	})

	t.Run("should retry indefinitely until context is cancelled with retryCount = 0", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		iTime := MockTime
		retryDelay := time.Millisecond
		retryCount := 0
		var attempts int32 = 0
		f := func() error {
			atomic.AddInt32(&attempts, 1)
			if atomic.LoadInt32(&attempts) == 2 {
				cancel()
			}
			iTime.Sleep(retryDelay + time.Millisecond)
			return errors.New("temporary error")
		}

		err := Retry(ctx, iTime, retryDelay, retryCount, f)
		require.Error(t, err)
	})
}
