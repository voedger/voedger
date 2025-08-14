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

var testCfg = NewConfig(100*time.Millisecond, 5*time.Second)

func TestNewConfig(t *testing.T) {
	require := require.New(t)
	baseDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second
	cfg := NewConfig(baseDelay, maxDelay)
	require.Equal(baseDelay, cfg.BaseDelay)
	require.Equal(maxDelay, cfg.MaxDelay)
}

func TestInvalidConfig(t *testing.T) {
	testCases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "negative base delay",
			cfg: Config{
				BaseDelay: -100 * time.Millisecond,
				MaxDelay:  1 * time.Second,
			},
		},
		{
			name: "zero base delay",
			cfg: Config{
				BaseDelay: 0,
				MaxDelay:  1 * time.Second,
			},
		},
		{
			name: "negative max delay",
			cfg: Config{
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  -1 * time.Second,
			},
		},
		{
			name: "zero max delay",
			cfg: Config{
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  0,
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

	t.Run("initially cancelled", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("permanent error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := Retry(ctx, testCfg, fn)

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

		result, err := Retry(ctx, testCfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Empty(result)
	})

	t.Run("cancel in operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		fn := func() (string, error) {
			cancel()
			return "", errors.New("permanent error")
		}

		result, err := Retry(ctx, testCfg, fn)

		require.ErrorIs(err, context.Canceled)
		require.Empty(result)
	})
}

func TestNextDelay_GrowsOnAverage(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		BaseDelay:               10 * time.Millisecond,
		MaxDelay:                800 * time.Millisecond, // small cap to keep test fast
		ResetDelayAfterMaxDelay: false,                  // avoid internal resets affecting attempt
	}
	r, err := New(cfg)
	require.NoError(err)

	const samplesPerAttempt = 1000
	var prevMean float64
	var prevCap float64

	for attempt := 0; attempt < 50; attempt++ {
		// Compute the expected cap: min(base * 2^attempt, max)
		cap := float64(cfg.BaseDelay) * float64(uint64(1)<<attempt) // nolint predeclared
		if cap > float64(cfg.MaxDelay) {
			cap = float64(cfg.MaxDelay)
		}

		// Sample many delays at this attempt level.
		var sum float64
		for i := 0; i < samplesPerAttempt; i++ {
			// Force the attempt we want to test (NextDelay mutates it).
			r.attempt = attempt
			d := r.NextDelay()

			// Basic bounds: 0 <= d < cap
			require.GreaterOrEqual(d, time.Duration(0), "Delay %d should be >= 0", i)
			if float64(d) >= cap && cap > 0 { // Duration truncates, so equality shouldn't happen
				t.Fatalf("delay exceeds cap at attempt=%d: d=%v cap=%v", attempt, d, time.Duration(cap))
			}
			sum += float64(d)
		}
		mean := sum / samplesPerAttempt

		// Only check monotonic growth when the cap itself increases.
		if attempt > 0 && cap > prevCap {
			// Because mean ~ cap/2 and samplesPerAttempt is large,
			// the mean should clearly increase when cap increases.
			require.Greater(mean, prevMean, "mean delay did not increase: attempt=%d prevMean=%.2f mean=%.2f prevCap=%v cap=%v",
				attempt, prevMean, mean, time.Duration(prevCap), time.Duration(cap))

			// Be stricter: demand a noticeable increase relative to noise.
			// When cap doubles, mean should ~double; require at least +25% to avoid flakiness.
			if mean < prevMean*1.25 && cap >= prevCap*1.5 {
				t.Fatalf("mean delay increase too small: attempt=%d prevMean=%.2f mean=%.2f prevCap=%v cap=%v",
					attempt, prevMean, mean, time.Duration(prevCap), time.Duration(cap))
			}
		}

		prevMean = mean
		prevCap = cap

		// Once the cap has saturated at MaxDelay, we can stop early.
		if cap == float64(cfg.MaxDelay) {
			// Optional: verify plateau (means should be ~stable around MaxDelay/2)
			// but we won't enforce exact value to avoid crypto/rand variance edge cases.
			break
		}
	}
}

func TestResetDelayAfterMaxDelay(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		BaseDelay:               10 * time.Millisecond,
		MaxDelay:                80 * time.Millisecond, // base * 2^3 = 80ms hits the cap
		ResetDelayAfterMaxDelay: true,
	}
	r, err := New(cfg)
	require.NoError(err)

	// Force the attempt so that exponentialDelay = base * 2^attempt = 80ms == MaxDelay.
	// With ResetDelayAfterMaxDelay == true, this call should reset r.attempt to 0.
	r.attempt = 3
	_ = r.NextDelay()

	require.Zero(r.attempt, "attempt did not reset after hitting cap; got %d, want 0", r.attempt)

	// Next call should use attempt=0, i.e., cap = BaseDelay.
	d := r.NextDelay()
	if d < 0 || d >= cfg.BaseDelay {
		t.Fatalf("delay not reset to base range; got %v, want in [0, %v)", d, cfg.BaseDelay)
	}
}

func TestMaxDelayCapping(t *testing.T) {
	require := require.New(t)
	cfg := Config{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  200 * time.Millisecond,
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

	cfg.OnError = func(attempt int, delay time.Duration, _ error) (bool, error) {
		retryDelays = append(retryDelays, delay)
		return true, nil
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

	fn := func() (string, error) {
		return "immediate success", nil
	}

	result, err := Retry(context.Background(), testCfg, fn)
	require.NoError(err)
	require.Equal("immediate success", result)
}

func TestOnError(t *testing.T) {
	require := require.New(t)
	testErr := errors.New("test error")
	cfg := NewConfig(100*time.Millisecond, 3*time.Second)

	t.Run("Retry", func(t *testing.T) {
		retriesNum := 0
		fn := func() error {
			switch retriesNum {
			case 0, 1, 2:
				return testErr
			}
			return nil
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) (bool, error) {
			retriesNum++
			return true, nil
		}
		err := RetryNoResult(context.Background(), cfg, fn)
		require.NoError(err)
		require.Equal(3, retriesNum)
	})

	t.Run("Accept", func(t *testing.T) {
		retriesNum := 0
		fn := func() error {
			return testErr
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) (bool, error) {
			retriesNum++
			if retriesNum == 1 {
				return true, nil
			}
			return false, nil
		}
		err := RetryNoResult(context.Background(), cfg, fn)
		require.NoError(err)
		require.Equal(2, retriesNum)
	})

	t.Run("Fail fast", func(t *testing.T) {
		retriesNum := 0
		fn := func() error {
			return testErr
		}
		cfg.OnError = func(attempt int, delay time.Duration, err error) (bool, error) {
			retriesNum++
			if retriesNum == 1 {
				return true, nil
			}
			return false, err
		}
		err := RetryNoResult(context.Background(), cfg, fn)
		require.ErrorIs(err, testErr)
		require.Equal(2, retriesNum)
	})
}
