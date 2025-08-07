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
	"errors"
	"time"
)

func NewDefaultConfig() Config {
	return Config{
		JitterFactor: DefaultJitterFactor,
		Multiplier:   DefaultMultiplier,
	}
}

// New creates a Retrier with provided Config, validating parameters.
func New(cfg Config) (*Retrier, error) {
	if cfg.InitialInterval <= 0 || cfg.MaxInterval < 0 || (cfg.MaxInterval == 0 && cfg.Multiplier != 1) ||
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
	if r.cfg.MaxInterval > 0 && next > r.cfg.MaxInterval {
		next = r.cfg.MaxInterval
	}
	r.currentInterval = next

	// apply FullJitter: random offset around base
	// offset in [-JitterFactor*base, +JitterFactor*base]
	offset := (secureFloat64()*2 - 1) * r.cfg.JitterFactor * float64(base)
	delay := max(base+time.Duration(offset), 0)
	return delay
}

// Run executes fn until it succeeds, matches Acceptable errors, or context is done/other error.
func (r *Retrier) Run(ctx context.Context, fn func() error) error {
	attempt := 0

	for ctx.Err() == nil {
		err := fn()

		// success on nil
		if err == nil {
			return nil
		}

		// treat specified Acceptable errors as success
		for _, ae := range r.cfg.Acceptable {
			if errors.Is(err, ae) {
				return nil
			}
		}

		// decide whether to retry
		retry := false
		if len(r.cfg.RetryOnlyOn) > 0 {
			for _, re := range r.cfg.RetryOnlyOn {
				if errors.Is(err, re) {
					retry = true
					break
				}
			}
		} else {
			// default: retry all errors except context
			retry = true
		}

		if !retry {
			// abort immediately with this error
			return err
		}

		// prepare backoff
		attempt++

		delay := r.NextDelay()
		// OnRetry callback
		if r.cfg.OnRetry != nil {
			r.cfg.OnRetry(attempt, delay, err)
		}

		// wait for delay or context done
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
