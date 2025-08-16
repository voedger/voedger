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
	"math"
	"math/bits"
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
	var prevMean time.Duration
	var prevCap time.Duration

	for attempt := 0; attempt < 50; attempt++ {
		// Compute the expected cap: min(base * 2^attempt, max)
		delayCap := cfg.BaseDelay << attempt
		if delayCap < 0 || delayCap > cfg.MaxDelay { // overflow or max reached
			delayCap = cfg.MaxDelay
		}

		// Sample many delays at this attempt level.
		var sum time.Duration
		for i := 0; i < samplesPerAttempt; i++ {
			// Force the attempt we want to test (NextDelay mutates it).
			r.attempt = attempt
			d := r.NextDelay()

			// Basic bounds: 0 <= d < delayCap
			require.GreaterOrEqual(d, time.Duration(0), "Delay %d should be >= 0", i)
			if d >= delayCap && delayCap > 0 {
				t.Fatalf("delay exceeds cap at attempt=%d: d=%v cap=%v", attempt, d, delayCap)
			}
			sum += d
		}
		mean := sum / samplesPerAttempt // integer mean in Duration

		// Only check monotonic growth when the cap itself increases.
		if attempt > 0 && delayCap > prevCap {
			// Because mean ~ delayCap/2 and samplesPerAttempt is large,
			// the mean should clearly increase when delayCap increases.
			require.Greater(mean, prevMean,
				"mean delay did not increase: attempt=%d prevMean=%v mean=%v prevCap=%v cap=%v",
				attempt, prevMean, mean, prevCap, delayCap)

			// Be stricter: demand a noticeable increase (~+25%) when cap grows by >=50%.
			if mean*100 < prevMean*125 && delayCap >= prevCap*3/2 {
				t.Fatalf("mean delay increase too small: attempt=%d prevMean=%v mean=%v prevCap=%v cap=%v",
					attempt, prevMean, mean, prevCap, delayCap)
			}
		}

		prevMean = mean
		prevCap = delayCap

		// Once the cap has saturated at MaxDelay, we can stop early.
		if delayCap == cfg.MaxDelay {
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

func TestNextDelayOverflowAverageIsConstant2(t *testing.T) {
	require := require.New(t)
	// We’ll force the overflow branch: attempt >= bits.UintSize-1.
	// In that branch NextDelay() draws U[0, MaxDelay) and (since ResetDelayAfterMaxDelay==true)
	// resets attempt to 0; we’ll push it back to overflow before each sample.
	const (
		samples   = 20000 // crypto/rand-backed; keep reasonably large but quick
		buckets   = 10
		baseDelay = 7 * time.Millisecond
		maxDelay  = 1 * time.Second
	)

	cfg := NewConfig(baseDelay, maxDelay)
	r, err := New(cfg)
	require.NoError(err)

	overflowAttempt := bits.UintSize - 1

	var sum float64
	bucketSums := make([]float64, buckets)
	bucketCounts := make([]int, buckets)

	for i := 0; i < samples; i++ {
		// Force the overflow path on every draw.
		r.attempt = overflowAttempt

		d := r.NextDelay()
		val := float64(d)

		sum += val
		b := i / (samples / buckets) // Fix: even bucket distribution
		if b >= buckets {
			b = buckets - 1
		}
		bucketSums[b] += val
		bucketCounts[b]++
	}

	// Expected mean for U[0, MaxDelay) is MaxDelay/2.
	expMean := float64(maxDelay) / 2.0
	mean := sum / float64(samples)

	// 1) Check the overall mean is close to MaxDelay/2.
	// Allow a small relative error; with 20k samples from a uniform, 5% is very safe.
	// Check both upper and lower bounds
	const relTol = 0.05
	require.GreaterOrEqual(mean, expMean*(1-relTol), "mean too low: got %.3fms, want ~%.3fms (±%.1f%%)",
		mean/float64(time.Millisecond), expMean/float64(time.Millisecond), relTol*100)
	require.LessOrEqual(mean, expMean*(1+relTol), "mean too high: got %.3fms, want ~%.3fms (±%.1f%%)",
		mean/float64(time.Millisecond), expMean/float64(time.Millisecond), relTol*100)

	// 2) Check for “no growth” across time: bucket means should be stable,
	// and the last bucket should not be meaningfully larger than the first.
	bucketMeans := make([]float64, buckets)
	for i := range bucketMeans {
		require.NotZero(bucketCounts[i], "bucket %d empty", i)
		bucketMeans[i] = bucketSums[i] / float64(bucketCounts[i])
	}

	first := bucketMeans[0]
	last := bucketMeans[len(bucketMeans)-1]

	// Let buckets vary by noise, but forbid any clear upward trend:
	// last cannot exceed first by more than 10% of expected mean.
	// Check for both upward and downward trends
	const trendTol = 0.10
	require.LessOrEqual(math.Abs(last-first), trendTol*expMean,
		"trend detected: first=%.3fms last=%.3fms diff=%.3fms (limit=±%.1f%% of exp mean)",
		first/float64(time.Millisecond), last/float64(time.Millisecond),
		math.Abs(last-first)/float64(time.Millisecond), trendTol*100)

	// Also enforce all bucket means stay within a reasonable band around expMean.
	// For uniform noise, ±12.5% is lenient and avoids flakiness on slow CI.
	const band = 0.125
	low, high := expMean*(1-band), expMean*(1+band)
	for i, m := range bucketMeans {
		require.GreaterOrEqual(m, low, "bucket %d mean too low: got %.3fms, want >= %.3fms",
			i, m/float64(time.Millisecond), low/float64(time.Millisecond))
		require.LessOrEqual(m, high, "bucket %d mean too high: got %.3fms, want <= %.3fms",
			i, m/float64(time.Millisecond), high/float64(time.Millisecond))
	}
}

func TestSaturatingMulPow2(t *testing.T) {
	req := require.New(t)

	const (
		maxDur       = time.Duration(math.MaxInt64)
		shiftSignBit = 63 // int64 sign bit; Duration is always int64
	)

	tests := []struct {
		name     string
		input    time.Duration
		shift    int
		expected time.Duration
	}{
		// Zero and negative inputs / shifts
		{"zero input", 0, 5, 0},
		{"negative input", -100 * time.Millisecond, 3, -100 * time.Millisecond},
		{"zero shift", 100 * time.Millisecond, 0, 100 * time.Millisecond},
		{"negative shift", 100 * time.Millisecond, -1, 100 * time.Millisecond},
		{"very negative shift", 250 * time.Millisecond, math.MinInt, 250 * time.Millisecond},

		// Normal doubling
		{"shift 1", 100 * time.Millisecond, 1, 200 * time.Millisecond},
		{"shift 2", 100 * time.Millisecond, 2, 400 * time.Millisecond},
		{"shift 3", 1 * time.Second, 3, 8 * time.Second},

		// Saturation threshold tied to int64, not bits.UintSize
		{"limit-1 ok (62)", 1 * time.Nanosecond, 62, time.Duration(1 << 62)},
		{"at sign bit (63) saturates", 1 * time.Nanosecond, 63, maxDur},
		{"beyond sign bit (64) saturates", 1 * time.Nanosecond, 64, maxDur},

		// Overflow detection around boundary
		{"large base overflow", time.Duration(math.MaxInt64 / 2), 2, maxDur},
		{"boundary exact", time.Duration(math.MaxInt64 / 4), 2, time.Duration(math.MaxInt64/4) * 4},
		{"over boundary", time.Duration(math.MaxInt64/4 + 1), 2, maxDur},

		// Edge around huge factor (1<<62)
		{"exact at 1<<62", time.Duration(math.MaxInt64 >> 62), 62, time.Duration(math.MaxInt64>>62) * (1 << 62)},
		{"just over at 1<<62", time.Duration((math.MaxInt64 >> 62) + 1), 62, maxDur},

		// Max values
		{"max int64 input", time.Duration(math.MaxInt64), 1, maxDur},
		{"over half max with shift 1", time.Duration(math.MaxInt64/2 + 1), 1, maxDur},

		// Small values with larger shifts
		{"1ns shift 1", 1 * time.Nanosecond, 1, 2 * time.Nanosecond},
		{"1ns shift 10", 1 * time.Nanosecond, 10, 1024 * time.Nanosecond},
		{"1ns shift 20", 1 * time.Nanosecond, 20, time.Duration(1 << 20)},
		{"1ns shift 30", 1 * time.Nanosecond, 30, time.Duration(1 << 30)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := saturatingMulPow2(tt.input, tt.shift)
			req.Equal(tt.expected, got)
		})
	}

	// Idempotence once saturated
	t.Run("saturated stays saturated", func(t *testing.T) {
		x := time.Duration(math.MaxInt64/2 + 1)
		req.Equal(maxDur, saturatingMulPow2(x, 1))
		req.Equal(maxDur, saturatingMulPow2(x, 10))
		req.Equal(maxDur, saturatingMulPow2(x, 100))
	})

	// Monotonicity in shift for positive x until saturation
	t.Run("monotonic nondecreasing in s", func(t *testing.T) {
		x := 3 * time.Millisecond
		prev := saturatingMulPow2(x, 0)
		for s := 1; s < shiftSignBit+5; s++ {
			cur := saturatingMulPow2(x, s)
			req.GreaterOrEqual(cur, prev)
			if prev == maxDur {
				req.Equal(maxDur, cur)
			}
			prev = cur
		}
	})

	// Negative input remains unchanged even for huge shift
	t.Run("negative input unchanged for huge shift", func(t *testing.T) {
		req.Equal(-5*time.Second, saturatingMulPow2(-5*time.Second, 1000))
	})
}
