/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package elections

import (
	"context"
	"sync"

	"github.com/voedger/voedger/pkg/coreutils"
)

type elections[K comparable, V any] struct {
	storage     ITTLStorage[K, V]
	clock       coreutils.ITime
	mu          sync.Mutex
	isFinalized bool
	leadership  map[K]*leaderInfo[K, V]
}

// leaderInfo holds per-key tracking data for a leadership.
type leaderInfo[K comparable, V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // used to wait for the renewal goroutine
}
