/*
 * Copyright (c) 2025-present unTill Pro, Ltd. and Contributors
 * @author Denis Gribanov
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package retrier_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	retrier "github.com/voedger/voedger/pkg/goutils/retry"
)

func ExampleRetry() {
	cfg := retrier.Config{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.5,
	}

	attempts := 0
	result, err := retrier.Retry(context.Background(), cfg, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	})

	fmt.Printf("Result: %s\n", result)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Attempts: %d\n", attempts)
	// Output:
	// Result: success
	// Error: <nil>
	// Attempts: 3
}

func ExampleRetryErr() {
	cfg := retrier.Config{
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.5,
	}

	attempts := 0
	err := retrier.RetryErr(context.Background(), cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Attempts: %d\n", attempts)
	// Output:
	// Error: <nil>
	// Attempts: 3
}

func ExampleRetryFor() {
	// 1) Configure a fast backoff for demonstration:
	cfg := retrier.Config{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     300 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
		ResetAfter:      0, // no reset
	}

	// 2) Create a context with a deadline of 1 second
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 3) Simulate an operation that fails twice before succeeding
	attempts := 0
	ok, err := retrier.RetryFor(ctx, cfg, 800*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			fmt.Printf("Attempt %d: temporary error\n", attempts)
			return errors.New("temporary")
		}
		fmt.Printf("Attempt %d: success\n", attempts)
		return nil
	})

	if err != nil {
		fmt.Printf("final error: %v\n", err)
	} else if !ok {
		fmt.Println("did not succeed within timeout")
	} else {
		fmt.Printf("succeeded after %d attempts\n", attempts)
	}

	// Output:
	// Attempt 1: temporary error
	// Attempt 2: temporary error
	// Attempt 3: success
	// succeeded after 3 attempts
}
