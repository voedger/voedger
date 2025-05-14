/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package ielections

import (
	"sync"

	"github.com/voedger/voedger/pkg/goutils/timeu"
)

// Provide constructs an IElections[K,V] instance using the provided storage and clock.
// It returns the IElections[K,V] instance and a cleanup function that should be called when done.
// cleanup function waits for all goroutines to finish
func Provide[K any, V any](storage ITTLStorage[K, V], clock timeu.ITime) (IElections[K, V], func()) {
	elector := &elections[K, V]{
		storage:    storage,
		clock:      clock,
		leadership: sync.Map{},
	}

	return elector, elector.cleanup
}
