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
	"encoding/binary"
	"math/bits"
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
	r := &Retrier{
		cfg:       cfg,
		lastReset: time.Now(),
	}
	return r, nil
}

// NextDelay computes the next delay using [Full Jitter algorithm from AWS](https://aws.amazon.com/ru/blogs/architecture/exponential-backoff-and-jitter/):
// sleep = random_between(0, min(cap, base*2^attempt))
func (r *Retrier) NextDelay() time.Duration {
	const maxSafeShift = bits.UintSize - 1 // 63 on 64-bit, 31 on 32-bit
	exponentialDelay := float64(r.cfg.MaxDelay)
	if r.attempt < maxSafeShift {
		// Calculate exponential backoff: base*2^attempt
		exponentialDelay = float64(r.cfg.BaseDelay) * float64(uint64(1)<<r.attempt)
	}

	// Apply max delay cap
	cap := min(exponentialDelay, float64(r.cfg.MaxDelay)) // nolint predeclared

	// Full Jitter: uniform random in [0, cap]
	delay := time.Duration(secureFloat64() * cap)

	if r.cfg.ResetDelayAfterMaxDelay && exponentialDelay >= float64(r.cfg.MaxDelay) {
		r.attempt = 0
	} else {
		r.attempt++
	}

	return delay
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

// secureFloat64 returns a cryptographically secure random float64 in the range [0, 1).
func secureFloat64() float64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		// notest
		panic(err)
	}
	// Convert the random bytes to a uint64
	u := binary.LittleEndian.Uint64(buf[:])
	// Convert to float64 in [0, 1) by dividing by 2^64
	return float64(u) / float64(1<<64)
}
