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
	"time"
)

func NewConfigConstantBackoff(initialDelay time.Duration) Config {
	return Config{
		InitialDelay: initialDelay,
		Multiplier:   1,
	}
}

func NewConfigExponentialBackoff(initialDelay time.Duration, maxDelay time.Duration) Config {
	return Config{
		JitterFactor: DefaultJitterFactor,
		Multiplier:   DefaultMultiplier,
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
	}
}

// New creates a Retrier with provided Config, validating parameters.
func New(cfg Config) (*Retrier, error) {
	if cfg.InitialDelay <= 0 || cfg.MaxDelay < 0 || (cfg.MaxDelay == 0 && cfg.Multiplier > 1) ||
		cfg.Multiplier < 1 || cfg.JitterFactor < 0 || cfg.JitterFactor > 1 ||
		cfg.ResetAfter < 0 {
		return nil, ErrInvalidConfig
	}
	r := &Retrier{
		cfg:          cfg,
		currentDelay: cfg.InitialDelay,
		lastReset:    time.Now(),
	}
	return r, nil
}

// NextDelay computes the next delay, applying exponential backoff,
// FullJitter, and reset logic.
func (r *Retrier) NextDelay() time.Duration {
	now := time.Now()
	// reset delay if period elapsed
	if r.cfg.ResetAfter > 0 && now.Sub(r.lastReset) >= r.cfg.ResetAfter {
		r.currentDelay = r.cfg.InitialDelay
		r.lastReset = now
	}
	// compute base delay
	base := r.currentDelay
	// prepare next delay for future

	next := time.Duration(float64(base) * r.cfg.Multiplier)
	if r.cfg.MaxDelay > 0 && next > r.cfg.MaxDelay {
		next = r.cfg.MaxDelay
	}
	r.currentDelay = next

	// apply FullJitter: random offset around base
	// offset in [-JitterFactor*base, +JitterFactor*base]
	offset := (secureFloat64()*2 - 1) * r.cfg.JitterFactor * float64(base)
	delay := max(base+time.Duration(offset), 0)
	return delay
}

func (r *Retrier) Run(ctx context.Context, fn func() error) error {
	attempt := 0

	for ctx.Err() == nil {

		err := fn()

		if err == nil {
			return nil
		}

		// backoff computation
		attempt++
		delay := r.NextDelay()

		action := DoRetry
		if r.cfg.OnError != nil {
			action = r.cfg.OnError(attempt, delay, err)
		}

		switch action {
		case Accept:
			return nil
		case Abort:
			return err
		case DoRetry:
		default:
			panic("unknown retry action")
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
