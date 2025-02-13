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
	storage       ITTLStorage[K, V]
	leadership    map[K]*leaderInfo[K, V]
	clock         coreutils.ITime
	mu            sync.Mutex
	isFinalized   bool
	finalizeMutex sync.Mutex
	wg            sync.WaitGroup
}

// leaderInfo holds per-key tracking data for a leadership.
type leaderInfo[K comparable, V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	// renewalIsStarted chan struct{}
}
