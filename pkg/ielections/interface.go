/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"context"
)

// IElections has AcquireLeadership returning nil if leadership is not acquired or error.
// [~server.design.orch/IElections~impl]
type IElections[K any, V any] interface {
	// AcquireLeadership attempts to become leader for `key` with `val`.
	//  - Returns a non-nil context if leadership is acquired successfully.
	//  - Returns nil if leadership cannot be acquired or an error occurs.
	// The background goroutine is spawned only on success.
	AcquireLeadership(key K, val V, leadershipDurationSecods LeadershipDurationSeconds) (ctx context.Context)

	// ReleaseLeadership stops the background renewal goroutine for `key` and wait till it finished
	// CompareAndDeletes from storage if we still hold it. No return value.
	ReleaseLeadership(key K)
}

// ITTLStorage defines a TTL-based storage layer with explicit durations.
// [~server.design.orch/ITTLStorage~impl]
type ITTLStorage[K any, V any] interface {
	// InsertIfNotExist tries to insert (key, val) with a TTL only if key does not exist.
	// Returns (true, nil) if inserted successfully,
	// (false, nil) if the key already exists,
	// or (false, err) if a storage error occurs.
	InsertIfNotExist(key K, val V, ttlSeconds int) (bool, error)

	// CompareAndSwap checks if the current value for `key` is `oldVal`.
	// If it matches, sets it to `newVal` and updates the TTL to `ttl`.
	// Returns (true, nil) on success, (false, nil) if values do not match, or (false, err) on error.
	CompareAndSwap(key K, oldVal V, newVal V, ttlSeconds int) (bool, error)

	// CompareAndDelete compares the current value for `key` with `val`,
	// and if they match, deletes the key, returning (true, nil). Otherwise, (false, nil).
	// On storage error, returns (false, err).
	CompareAndDelete(key K, val V) (bool, error)

	// used in tests only
	// actual TTL checking depends on driver
	Get(key K) (ok bool, val V, err error)
}
