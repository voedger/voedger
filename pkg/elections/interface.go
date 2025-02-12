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
	InsertIfNotExist(key K, val V) bool

	// CompareAndSwap compares the current value for key with oldVal;
	// if they match, it swaps the value to newVal and returns true.
	// Otherwise, returns false.
	CompareAndSwap(key K, oldVal V, newVal V) bool

	// CompareAndDelete compares the current value for key with val,
	// and if they match, it deletes the key and returns true;
	// otherwise, returns false.
	CompareAndDelete(key K, val V) bool
}

// IElections defines the leader-election interface.
type IElections[K comparable, V any] interface {
	// AcquireLeadership attempts to become the leader for `key` with value `val`,
	// for at most `duration`. If successful, returns a context that remains
	// active while we hold leadership. If leadership fails or is immediately
	// unavailable, a canceled context is returned. Any errors are logged.
	AcquireLeadership(key K, val V, duration time.Duration) (ctx context.Context)

	// ReleaseLeadership releases the leader position for `key` immediately,
	// stops the background renewal goroutine for this key, and cleans up storage.
	ReleaseLeadership(key K) error
}
