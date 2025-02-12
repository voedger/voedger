/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
	"time"
)

// ITTLStorage defines the minimal TTL-based storage required by the elections.
type ITTLStorage[K comparable, V any] interface {
	// InsertIfNotExist tries to insert (key, val) only if key does not exist
	// and returns true if inserted successfully; false otherwise.
	InsertIfNotExist(key K, val V) (bool, error)

	// CompareAndSwap compares the current value for key with oldVal;
	// if they match, it swaps the value to newVal and returns true.
	// Otherwise, returns false.
	CompareAndSwap(key K, oldVal V, newVal V) (bool, error)

	// CompareAndDelete compares the current value for key with val,
	// and if they match, it deletes the key and returns true;
	// otherwise, returns false.
	CompareAndDelete(key K, val V) (bool, error)
}

// IElections has AcquireLeadership returning nil if leadership is not acquired or error.
type IElections[K comparable, V any] interface {
	// AcquireLeadership attempts to become leader for `key` with `val`.
	//  - Returns a non-nil context if leadership is acquired successfully.
	//  - Returns nil if leadership cannot be acquired or an error occurs.
	// The background goroutine is spawned only on success.
	AcquireLeadership(key K, val V, duration time.Duration) (ctx context.Context)

	// ReleaseLeadership stops the background renewal goroutine for `key` and
	// CompareAndDeletes from storage if we still hold it. No return value.
	ReleaseLeadership(key K)
}
