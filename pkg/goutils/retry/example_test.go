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

func ExampleRetry_handleTransientFailures() {
	cfg := retrier.NewConfig(100*time.Millisecond, 3*time.Second)

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

func ExampleRetry_acceptExpectedErrors() {
	cfg := retrier.NewConfig(100*time.Millisecond, 3*time.Second)

	attempts := 0
	acceptableErr := errors.New("acceptable error")
	cfg.OnError = func(attempt int, delay time.Duration, opErr error) (retry bool, err error) {
		attempts++
		if errors.Is(opErr, acceptableErr) {
			return false, nil
		}
		return true, opErr
	}

	res, err := retrier.Retry(context.Background(), cfg, func() (int, error) {
		return 42, acceptableErr
	})
	fmt.Printf("Result: %v\n", res)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Attempts: %d\n", attempts)

	// Output:
	// Result: 42
	// Error: <nil>
	// Attempts: 1
}

func ExampleRetry_failFastOnFatalError() {
	cfg := retrier.NewConfig(100*time.Millisecond, 3*time.Second)

	attempts := 0
	fatalErr := errors.New("fatal error")
	cfg.OnError = func(attempt int, delay time.Duration, opErr error) (retry bool, err error) {
		attempts++
		if errors.Is(opErr, fatalErr) {
			return false, opErr
		}
		return true, opErr
	}

	res, err := retrier.Retry(context.Background(), cfg, func() (int, error) {
		return 0, fatalErr
	})
	fmt.Printf("Result: %v\n", res)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Attempts: %d\n", attempts)

	// Output:
	// Result: 0
	// Error: fatal error
	// Attempts: 1
}

func ExampleRetryNoResult() {
	cfg := retrier.NewConfig(100*time.Millisecond, 3*time.Second)

	attempts := 0
	err := retrier.RetryNoResult(context.Background(), cfg, func() error {
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
