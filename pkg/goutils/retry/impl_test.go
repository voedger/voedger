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

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestDefaultConfig(t *testing.T) {
	require := require.New(t)
	cfg := NewDefaultConfig()
	require.Equal(0.5, cfg.JitterFactor)
	require.Equal(2.0, cfg.Multiplier)
}

func TestInvalidConfig(t *testing.T) {
	testCases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "negative initial interval",
			cfg: Config{
				InitialInterval: -100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.5,
			},
		},
		{
			name: "zero initial interval",
			cfg: Config{
				InitialInterval: 0,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.5,
			},
		},
		{
			name: "negative max interval",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     -1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.5,
			},
		},
		{
			name: "zero max interval",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     0,
				Multiplier:      2.0,
				JitterFactor:    0.5,
			},
		},
		{
			name: "multiplier less than 1",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      0.5,
				JitterFactor:    0.5,
			},
		},
		{
			name: "negative jitter factor",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    -0.1,
			},
		},
		{
			name: "jitter factor greater than 1",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    1.5,
			},
		},
		{
			name: "negative reset after",
			cfg: Config{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.5,
				ResetAfter:      -1 * time.Second,
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
			require.Equal("", result)
		})
	}
}

func TestContextCancellation(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}

	t.Run("initially cancelled", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("permanent error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Equal("", result)
	})

	t.Run("somewhen during error", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("permanent error")
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context after a short delay
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Equal("", result)
	})

	t.Run("right after op", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fn := func() (string, error) {
			cancel()
			return "", errors.New("permanent error")
		}

		result, err := Retry(ctx, cfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Equal("", result)
	})
}

func TestExponentialBackoffBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skip long test")
	}
	require := require.New(t)
	cfg := Config{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.0, // No jitter for deterministic testing
	}

	attempts := 0
	retryDelays := make([]time.Duration, 0)

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	// Track retry delays by using OnRetry callback
	cfg.OnRetry = func(attempt int, delay time.Duration) {
		retryDelays = append(retryDelays, delay)
	}

	ctx := context.Background()
	result, err := Retry(ctx, cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Equal(5, len(retryDelays)) // 5 retries, 5 delays

	// Verify exponential backoff behavior
	// Expected delays: ~100ms, ~200ms, ~400ms, ~800ms, ~1000ms (capped)
	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1000 * time.Millisecond, // Capped at max interval
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
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.0,
		ResetAfter:      200 * time.Millisecond,
	}

	attempts := 0
	retryDelays := make([]time.Duration, 0)

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	// Track retry delays by using OnRetry callback
	cfg.OnRetry = func(attempt int, delay time.Duration) {
		retryDelays = append(retryDelays, delay)
	}

	ctx := context.Background()
	result, err := Retry(ctx, cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Equal(5, len(retryDelays)) // 5 retries, 5 delays

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

func TestOnRetry(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}

	callbackCalls := 0

	cfg.OnRetry = func(attempt int, delay time.Duration) {
		callbackCalls++
	}

	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	ctx := context.Background()
	result, err := Retry(ctx, cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(2, callbackCalls)
}

func TestMaxIntervalCapping(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     200 * time.Millisecond, // Small max interval
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}

	attempts := 0
	retryDelays := make([]time.Duration, 0)

	fn := func() (string, error) {
		attempts++
		if attempts < 6 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	// Track retry delays by using OnRetry callback
	cfg.OnRetry = func(attempt int, delay time.Duration) {
		retryDelays = append(retryDelays, delay)
	}

	ctx := context.Background()
	result, err := Retry(ctx, cfg, fn)

	require.NoError(err)
	require.Equal("success", result)
	require.Equal(6, attempts)
	require.Equal(5, len(retryDelays)) // 5 retries, 5 delays

	for i, delay := range retryDelays {
		require.LessOrEqual(delay, cfg.MaxInterval,
			"Delay %d (%v) should not exceed max interval (%v)", i, delay, cfg.MaxInterval)
	}
}

func TestImmediateSuccess(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.5,
	}

	fn := func() (string, error) {
		return "immediate success", nil
	}

	ctx := context.Background()
	result, err := Retry(ctx, cfg, fn)

	require.NoError(err)
	require.Equal("immediate success", result)
}
