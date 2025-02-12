/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package elections

import "github.com/voedger/voedger/pkg/coreutils"

// Provide constructs an IElections[K,V] instance using the provided storage and clock.
// It returns the IElections[K,V] instance and a cleanup function that should be called when done.
func Provide[K comparable, V any](storage ITTLStorage[K, V], clock coreutils.ITime) (IElections[K, V], func()) {
	elector := &elections[K, V]{
		storage:    storage,
		clock:      clock,
		leadership: make(map[K]*leaderInfo[K, V]),
	}
	return elector, elector.cleanup
}
