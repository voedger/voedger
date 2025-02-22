/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import (
	"context"
)

// IElections has AcquireLeadership returning nil if leadership is not acquired or error.
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
