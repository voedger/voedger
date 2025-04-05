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
		// nolint
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
		retryCount := 0
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
		err := Retry_(ctx, MockTime, 0, f)

		// Verify results
		require.NoError(t, err)
		require.Equal(t, 1, attempts)
	})

	t.Run("retry and succeed after failures", func(t *testing.T) {
		// Setup test
		ctx := context.Background()
		var attempts int = 0
		f := func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		// Run test with goroutine for advancing mock time
		go func() {
			for attempts < 3 {
				MockTime.Add(10 * time.Millisecond)
				time.Sleep(time.Millisecond) // Give main goroutine time to process
			}
		}()

		// Call the function under test
		err := Retry_(ctx, MockTime, 5*time.Millisecond, f)

		// Verify results
		require.NoError(t, err)
		require.Equal(t, 3, attempts)
	})

	t.Run("context canceled during retry", func(t *testing.T) {
		// Setup test
		ctx, cancel := context.WithCancel(context.Background())
		var attempts int = 0
		f := func() error {
			attempts++
			if attempts == 2 {
				cancel()
			}
			return errors.New("persistent error")
		}

		// Run test with goroutine for advancing mock time
		go func() {
			for attempts < 2 {
				MockTime.Add(15 * time.Millisecond)
				time.Sleep(time.Millisecond) // Give main goroutine time to process
			}
		}()

		// Call the function under test
		err := Retry_(ctx, MockTime, 10*time.Millisecond, f)

		// Verify results
		require.Error(t, err)
		require.Equal(t, 2, attempts)
	})

	t.Run("context timeout", func(t *testing.T) {
		// Setup test
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		var attempts int = 0
		f := func() error {
			attempts++
			return errors.New("persistent error")
		}

		// Run test with goroutine for advancing mock time
		go func() {
			for attempts < 5 {
				MockTime.Add(15 * time.Millisecond)
				time.Sleep(time.Millisecond) // Give main goroutine time to process
			}
		}()

		// Call the function under test
		err := Retry_(ctx, MockTime, 10*time.Millisecond, f)

		// Verify results
		require.Error(t, err)
		require.Equal(t, 5, attempts)
	})

	t.Run("no delay stops after first failure", func(t *testing.T) {
		// Setup test
		ctx := context.Background()
		var attempts int = 0
		f := func() error {
			attempts++
			return errors.New("error on first attempt")
		}

		// Call the function under test
		err := Retry_(ctx, MockTime, 0, f)

		// Verify results
		require.Error(t, err)
		require.Equal(t, 1, attempts)
	})
}
