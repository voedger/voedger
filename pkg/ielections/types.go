/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package ielections

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/voedger/voedger/pkg/goutils/timeu"
)

type elections[K any, V any] struct {
	storage     ITTLStorage[K, V]
	leadership  sync.Map
	clock       timeu.ITime
	isFinalized atomic.Bool
}

// leaderInfo holds per-key tracking data for a leadership.
type leaderInfo[K any, V any] struct {
	val    V
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type LeadershipDurationSeconds int
