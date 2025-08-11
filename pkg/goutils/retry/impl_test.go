/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	require := require.New(t)
	initialDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second
	t.Run("constant", func(t *testing.T) {
		cfg := NewConfigConstantBackoff(initialDelay)
		require.Equal(initialDelay, cfg.InitialDelay)
		require.Zero(cfg.JitterFactor)
		require.EqualValues(1, cfg.Multiplier)
	})

	t.Run("exponential", func(t *testing.T) {
		cfg := NewConfigExponentialBackoff(initialDelay, maxDelay)
		require.Equal(initialDelay, cfg.InitialDelay)
		require.Equal(maxDelay, cfg.MaxDelay)
		require.Equal(0.5, cfg.JitterFactor)
		require.EqualValues(2, cfg.Multiplier)
	})
}

func TestInvalidConfig(t *testing.T) {
	testCases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "negative initial interval",
			cfg: Config{
				InitialDelay: -100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: 0.5,
			},
		},
		{
			name: "zero initial interval",
			cfg: Config{
				InitialDelay: 0,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: 0.5,
			},
		},
		{
			name: "negative max interval",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     -1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: 0.5,
			},
		},
		{
			name: "zero max interval when Multiplier != 1",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     0,
				Multiplier:   2.0,
				JitterFactor: 0.5,
			},
		},
		{
			name: "multiplier less than 1",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   0.5,
				JitterFactor: 0.5,
			},
		},
		{
			name: "negative jitter factor",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: -0.1,
			},
		},
		{
			name: "jitter factor greater than 1",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: 1.5,
			},
		},
		{
			name: "negative reset after",
			cfg: Config{
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
				Multiplier:   2.0,
				JitterFactor: 0.5,
				ResetAfter:   -1 * time.Second,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			fn := func() (string, error) {
				return "success", nil
			}
			ctx := context.Background()
			result, err := Retry(ctx, tc.cfg, fn)
			require.ErrorIs(err, ErrInvalidConfig)
			require.Empty(result)
		})
	}
}

func TestContextCancellation(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}

	t.Run("initially cancelled", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("permanent error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Empty(result)
	})

	t.Run("during retry", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("permanent error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Empty(result)
	})

	t.Run("cancel in operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fn := func() (string, error) {
			cancel()
			return "", errors.New("permanent error")
		}

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Empty(result)
	})
}

func TestExponentialBackoffBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skip long test")
	}
	require := require.New(t)
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}

	attempts := 0
	retryDelays := []time.Duration{}

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	cfg.OnError = func(attempt int, delay time.Duration, _ error) Action {
		retryDelays = append(retryDelays, delay)
		return DoRetry
	}

	result, err := Retry(context.Background(), cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Len(retryDelays, 5)

	// Verify exponential backoff behavior
	// Expected delays: ~100ms, ~200ms, ~400ms, ~800ms, ~1000ms (capped)
	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1000 * time.Millisecond,
	}

	for i, actualDelay := range retryDelays {
		expectedDelay := expectedDelays[i]
		// Allow some tolerance for timing variations
		tolerance := expectedDelay / 10 // 10% tolerance
		minDelay := expectedDelay - tolerance
		maxDelay := expectedDelay + tolerance

		require.GreaterOrEqual(actualDelay, minDelay, "Delay %d should be >= %v, got %v", i, minDelay, actualDelay)
		require.LessOrEqual(actualDelay, maxDelay, "Delay %d should be <= %v, got %v", i, maxDelay, actualDelay)
	}

	// Verify that delays are monotonically increasing (except for the last one which might be capped)
	for i := 1; i < len(retryDelays)-1; i++ {
		require.Greater(retryDelays[i], retryDelays[i-1],
			"Delay should increase exponentially: delay[%d]=%v, delay[%d]=%v",
			i, retryDelays[i], i-1, retryDelays[i-1])
	}
}

func TestResetAfter(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0,
		ResetAfter:   200 * time.Millisecond,
	}

	attempts := 0
	retryDelays := []time.Duration{}

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	// Track retry delays by using OnError callback
	cfg.OnError = func(attempt int, delay time.Duration, _ error) Action {
		retryDelays = append(retryDelays, delay)
		return DoRetry
	}

	result, err := Retry(context.Background(), cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Len(retryDelays, 5)

	// Verify that reset behavior is working
	// The key insight is that the reset happens when the time since last reset exceeds ResetAfter
	// Since we have no jitter, we can verify the pattern:

	// First delay should be initial interval
	require.Equal(100*time.Millisecond, retryDelays[0])

	// Second delay should be 2x initial
	require.Equal(200*time.Millisecond, retryDelays[1])

	// The third delay should show the reset behavior
	// If the reset is working, it should be 100ms (reset to initial)
	// If not working, it would be 400ms (2x previous)
	require.Equal(100*time.Millisecond, retryDelays[2],
		"Third delay should be reset to initial interval")

	// Fourth delay should be 2x the reset initial
	require.Equal(200*time.Millisecond, retryDelays[3])

	// Fifth delay should be reset to initial again (because 200ms has passed)
	require.Equal(100*time.Millisecond, retryDelays[4])
}

func TestMaxDelayCapping(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}

	attempts := 0
	retryDelays := []time.Duration{}

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	cfg.OnError = func(attempt int, delay time.Duration, _ error) Action {
		retryDelays = append(retryDelays, delay)
		return DoRetry
	}

	result, err := Retry(context.Background(), cfg, fn)
	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Len(retryDelays, 5)

	for _, d := range retryDelays {
		require.LessOrEqual(d, cfg.MaxDelay)
	}
}

func TestImmediateSuccess(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.5,
	}

	fn := func() (string, error) {
		return "immediate success", nil
	}

	result, err := Retry(context.Background(), cfg, fn)
	require.NoError(err)
	require.Equal("immediate success", result)
}

func TestOnError(t *testing.T) {
	require := require.New(t)
	testErr := errors.New("test error")

	t.Run("DoRetry", func(t *testing.T) {
		cfg := NewConfigConstantBackoff(100 * time.Millisecond)
		retriesNum := 0
		fn := func() error {
			switch retriesNum {
			case 0, 1, 2:
				return testErr
			}
			return nil
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) Action {
			retriesNum++
			return DoRetry
		}
		err := RetryErr(context.Background(), cfg, fn)
		require.NoError(err)
		require.Equal(3, retriesNum)
	})

	t.Run("Accept", func(t *testing.T) {
		cfg := NewConfigConstantBackoff(100 * time.Millisecond)
		retriesNum := 0
		fn := func() error {
			return testErr
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) Action {
			retriesNum++
			if retriesNum == 1 {
				return DoRetry
			}
			return Accept
		}
		err := RetryErr(context.Background(), cfg, fn)
		require.NoError(err)
		require.Equal(2, retriesNum)
	})

	t.Run("Abort", func(t *testing.T) {
		cfg := NewConfigConstantBackoff(100 * time.Millisecond)
		retriesNum := 0
		fn := func() error {
			return testErr
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) Action {
			retriesNum++
			if retriesNum == 1 {
				return DoRetry
			}
			return Abort
		}
		err := RetryErr(context.Background(), cfg, fn)
		require.ErrorIs(err, testErr)
		require.Equal(2, retriesNum)
	})

	t.Run("panic on unknown Action", func(t *testing.T) {
		cfg := NewConfigConstantBackoff(100 * time.Millisecond)
		fn := func() error { return testErr }
		cfg.OnError = func(attempt int, delay time.Duration, err error) Action {
			return -1
		}
		require.Panics(func() { RetryErr(context.Background(), cfg, fn) })
	})
}
