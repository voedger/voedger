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
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

func NewConfig(baseDelay time.Duration, maxDelay time.Duration) Config {
	return Config{
		BaseDelay:               baseDelay,
		MaxDelay:                maxDelay,
		ResetDelayAfterMaxDelay: true,
	}
}

// New creates a Retrier with provided Config, validating parameters.
func New(cfg Config) (*Retrier, error) {
	if cfg.BaseDelay <= 0 || cfg.MaxDelay <= 0 {
		return nil, ErrInvalidConfig
	}
	r := &Retrier{cfg: cfg}
	return r, nil
}

// NextDelay computes the next delay using [Full Jitter algorithm from AWS](https://aws.amazon.com/ru/blogs/architecture/exponential-backoff-and-jitter/):
// sleep = random_between(0, min(cap, base*2^attempt))
func (r *Retrier) NextDelay() time.Duration {

	// Saturating exponential: base * 2^attempt, capping at MaxInt64 on overflow.
	nextDelay := saturatingMulPow2(r.cfg.BaseDelay, r.attempt)

	// Apply cap
	nextDelay = min(nextDelay, r.cfg.MaxDelay)

	// Attempt accounting
	if r.cfg.ResetDelayAfterMaxDelay && nextDelay >= r.cfg.MaxDelay {
		r.attempt = 0
	} else {
		r.attempt++
	}

	// Full Jitter draw in [0, cap)
	if nextDelay > 0 {
		nextDelay = time.Duration(secureInt63n(int64(nextDelay)))
	}

	return nextDelay
}

func (r *Retrier) Run(ctx context.Context, fn func() error) error {
	r.attempt = 0 // Reset attempt counter at start
	totalAttempts := 0

	for ctx.Err() == nil {
		err := fn()
		if err == nil {
			return nil
		}

		// backoff computation
		totalAttempts++
		delay := r.NextDelay()

		if r.cfg.OnError != nil {
			retry, abortErr := r.cfg.OnError(totalAttempts, delay, err)
			if abortErr != nil {
				return abortErr
			}
			if !retry {
				return nil
			}
		}

		// context might have been cancelled while in OnError or in fn
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		// wait for back-off or cancellation
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

// saturatingMulPow2 returns x*2^s, saturating to MaxInt64 on overflow.
func saturatingMulPow2(x time.Duration, s int) time.Duration {
	const maxSafeShift = 63 // on 32-bit systems would be 31, that will cause too early saturation
	if x <= 0 || s <= 0 {
		return x
	}
	if s >= maxSafeShift { // would cross sign bit
		return math.MaxInt64
	}
	// Check if shift would overflow
	if x > math.MaxInt64>>s {
		return math.MaxInt64
	}
	return x << s
}

// secureInt63n returns a crypto-secure integer uniformly in [0, n).
func secureInt63n(n int64) int64 {
	x, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		// notest
		panic(err)
	}
	return x.Int64()
}
