/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package elections

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/voedger/voedger/pkg/coreutils"
)

type elections[K comparable, V any] struct {
	storage     ITTLStorage[K, V]
	leadership  sync.Map
	clock       coreutils.ITime
	isFinalized atomic.Bool
}

// leaderInfo holds per-key tracking data for a leadership.
type leaderInfo[K comparable, V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}
