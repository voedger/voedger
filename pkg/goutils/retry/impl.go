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
	"math/rand"
	"time"
)


func NewDefaultConfig() Config {
	return Config{
		JitterFactor: 0.5,
		Multiplier:   2,
	}
}

// New creates a Retrier with provided Config, validating parameters.
func New(cfg Config) (*Retrier, error) {
	if cfg.InitialInterval <= 0 || cfg.MaxInterval <= 0 ||
		cfg.Multiplier < 1 || cfg.JitterFactor < 0 || cfg.JitterFactor > 1 ||
		cfg.ResetAfter < 0 {
		return nil, ErrInvalidConfig
	}
	r := &Retrier{
		cfg:             cfg,
		currentInterval: cfg.InitialInterval,
		lastReset:       time.Now(),
	}
	return r, nil
}

// NextDelay computes the next delay, applying exponential backoff,
// FullJitter, and reset logic.
func (r *Retrier) NextDelay() time.Duration {
	now := time.Now()
	// reset interval if period elapsed
	if r.cfg.ResetAfter > 0 && now.Sub(r.lastReset) >= r.cfg.ResetAfter {
		r.currentInterval = r.cfg.InitialInterval
		r.lastReset = now
	}
	// compute base interval
	base := r.currentInterval
	// prepare next interval for future

	next := time.Duration(float64(base) * r.cfg.Multiplier)
	if next > r.cfg.MaxInterval {
		next = r.cfg.MaxInterval
	}
	r.currentInterval = next

	// apply FullJitter: random offset around base
	// offset in [-JitterFactor*base, +JitterFactor*base]
	offset := (rand.Float64()*2 - 1) * r.cfg.JitterFactor * float64(base)
	delay := base + time.Duration(offset)
	if delay < 0 {
		delay = 0
	}
	return delay
}

// Run retries operation until success or context cancellation.
func (r *Retrier) Run(ctx context.Context, operation func() error) error {
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := operation()
		if err == nil {
			return nil
		}
		attempt++
		// compute delay
		d := r.NextDelay()
		// callback before sleep
		if r.cfg.OnRetry != nil {
			r.cfg.OnRetry(attempt, d)
		}
		// wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
}

// Retry executes fn with retry logic and returns its result or an error.
func Retry[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	r, err := New(cfg)
	var zero T
	if err != nil {
		return zero, err
	}
	var result T
	err = r.Run(ctx, func() error {
		var fnErr error
		result, fnErr = fn()
		return fnErr
	})
	return result, err
}
