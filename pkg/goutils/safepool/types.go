/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package safepool

import (
	"sync"
	"sync/atomic"
)

type implPool[T any] struct {
	sync.Pool
	isStub       bool
	objectsInUse atomic.Uint64
	instantiator func(releaser IReleaser) T
}

type implIReleaser[T any] struct {
	obj                  T
	refCount             atomic.Uint64
	ownerPool            *implPool[T]
	isOwned              bool
	cleanupIntf          interface{ Cleanup() }
	borrowStackTrace     string
	initIntf             interface{ Init() }
	isInitIntfDetermined bool
	ownedMu              sync.Mutex
	ownedCond            *sync.Cond
	ownedInitializations uint64
	ownedTail            IReleaser
}

type stackFrame struct {
	fn   string
	file string
	line int
}

type stackTrace []stackFrame
