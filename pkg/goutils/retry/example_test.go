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