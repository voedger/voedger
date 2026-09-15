/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pool

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	m               sync.Mutex = sync.Mutex{}
	objectsCounters []func() uint64
	isDebug         atomic.Bool
	objAmounts      = map[string]int{}
)

func (st stackTrace) string() string {
	buf := bytes.NewBufferString("")
	for _, sf := range st {
		fmt.Fprintf(buf, "%s\n\t%s:%d\n", sf.fn, sf.file, sf.line)
	}
	return buf.String()
}

func (p *implPool[T]) Get() T {
	obj := p.get()
	releaseable := any(obj).(IReleaser)
	releaseable.reset()
	// Account for the borrow before Init, which may panic.
	p.objectsInUse.Add(1)
	if isDebug.Load() {
		st := getStackTrace().string()
		releaseable.setBorrowStackTrace(st)
		m.Lock()
		count := objAmounts[st]
		count++
		objAmounts[st] = count
		m.Unlock()
	}
	releaseable.init(obj)
	return obj
}

func (p *implPool[T]) get() T {
	var obj T
	if p.isStub {
		releaser := newReleaser(p)
		obj = p.instantiator(releaser)
		releaser.cleanupIntf, _ = any(obj).(interface{ Cleanup() })
		releaser.obj = obj
	} else {
		obj = p.Pool.Get().(T)
	}
	return obj
}

func (p *implPool[T]) GetOwned(owner IReleaser) T {
	if !owner.beginOwnedBorrow() {
		panic("owner already released")
	}
	defer owner.finishOwnedBorrow()

	obj := p.get()
	p.objectsInUse.Add(1)
	releaseable := any(obj).(IReleaser)
	releaseable.reset()
	releaseable.setIsOwned()
	owner.addOwned(releaseable)
	releaseable.init(obj)
	return obj
}

func (p *implPool[T]) GetObjectsInUse() uint64 {
	return p.objectsInUse.Load()
}

func (r *implIReleaser[T]) Release() {
	if r.isOwned {
		panic("must be released by owner")
	}
	r.releaseReference()
}

func (r *implIReleaser[T]) AddRef() {
	if r.isOwned {
		panic("cannot add reference to owned object")
	}
	for {
		refCount := r.refCount.Load()
		if refCount == 0 {
			panic("already released")
		}
		if refCount == ^uint64(0) {
			panic("reference count overflow")
		}
		if r.refCount.CompareAndSwap(refCount, refCount+1) {
			return
		}
	}
}

func (r *implIReleaser[T]) reset() {
	r.refCount.Store(1)
	r.isOwned = false
	r.borrowStackTrace = ""
}

func (r *implIReleaser[T]) IsOwned() bool {
	return r.isOwned
}

func (r *implIReleaser[T]) setIsOwned() {
	r.isOwned = true
}

func (r *implIReleaser[T]) setBorrowStackTrace(stackTrace string) {
	r.borrowStackTrace = stackTrace
}

func (r *implIReleaser[T]) releaseOwned() {
	r.releaseReference()
}

func (r *implIReleaser[T]) releaseReference() {
	for {
		refCount := r.refCount.Load()
		if refCount == 0 {
			panic("already released")
		}
		if !r.refCount.CompareAndSwap(refCount, refCount-1) {
			continue
		}
		if refCount > 1 {
			return
		}
		break
	}

	r.waitOwnedBorrows()
	if r.cleanupIntf != nil {
		r.cleanupIntf.Cleanup()
	}
	if ownedTail := r.takeOwnedTail(); ownedTail != nil {
		ownedTail.releaseOwned()
	}
	r.ownerPool.objectsInUse.Add(^uint64(0))
	// Remove the trace recorded for this borrow even if debug mode was
	// disabled after Get.
	if st := r.borrowStackTrace; st != "" {
		m.Lock()
		objAmounts[st]--
		if objAmounts[st] == 0 {
			delete(objAmounts, st)
		}
		m.Unlock()
		r.borrowStackTrace = ""
	}
	if !r.ownerPool.isStub {
		r.ownerPool.Put(r.obj)
	}
}

func (r *implIReleaser[T]) init(obj interface{}) {
	if !r.isInitIntfDetermined {
		r.initIntf, _ = obj.(interface{ Init() })
		r.isInitIntfDetermined = true
	}
	if r.initIntf != nil {
		r.initIntf.Init()
	}
}

func (r *implIReleaser[T]) beginOwnedBorrow() bool {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	if r.refCount.Load() == 0 {
		return false
	}
	r.ownedInitializations++
	return true
}

func (r *implIReleaser[T]) finishOwnedBorrow() {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	r.ownedInitializations--
	if r.ownedInitializations == 0 {
		r.ownedCond.Broadcast()
	}
}

func (r *implIReleaser[T]) addOwned(owned IReleaser) {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	owned.setOwnedTail(r.ownedTail)
	r.ownedTail = owned
}

func (r *implIReleaser[T]) waitOwnedBorrows() {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	for r.ownedInitializations > 0 {
		r.ownedCond.Wait()
	}
}

func (r *implIReleaser[T]) setOwnedTail(tail IReleaser) {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	r.ownedTail = tail
}

func (r *implIReleaser[T]) takeOwnedTail() IReleaser {
	r.ownedMu.Lock()
	defer r.ownedMu.Unlock()
	tail := r.ownedTail
	r.ownedTail = nil
	return tail
}

func newReleaser[T any](ownerPool *implPool[T]) *implIReleaser[T] {
	releaser := new(implIReleaser[T])
	releaser.ownerPool = ownerPool
	releaser.ownedCond = sync.NewCond(&releaser.ownedMu)
	return releaser
}

// NewPoolStub creates pool which does not act as a pool. I.e. just creates a new instance on each Get()
// Release() does nothing more but Cleanup() call if it exists
// does not track borrow source code points in debug mode
// useful for investigations
func NewPoolStub[T any](instantiator func(releaser IReleaser) T) IPool[T] {
	res := newPool(instantiator)
	res.instantiator = instantiator
	res.isStub = true
	return res
}

func NewPool[T any](instantiator func(releaser IReleaser) T) IPool[T] {
	res := newPool[T](nil)
	res.Pool = sync.Pool{
		New: func() interface{} {
			releaser := newReleaser(res)
			newInstance := instantiator(releaser)
			releaser.cleanupIntf, _ = any(newInstance).(interface{ Cleanup() })
			releaser.obj = newInstance
			return newInstance
		},
	}
	return res
}

func newPool[T any](instantiator func(releaser IReleaser) T) *implPool[T] {
	res := &implPool[T]{instantiator: instantiator}
	RegisterObjectsInUseCounter(func() uint64 { return res.GetObjectsInUse() })
	return res
}

func getStackTrace() stackTrace {
	const (
		maxStackDepth     = 100
		stackFramesToSkip = 3 // Skip runtime.Callers, getStackTrace, and implPool.Get.
	)
	pc := make([]uintptr, maxStackDepth)
	n := runtime.Callers(stackFramesToSkip, pc)
	frames := runtime.CallersFrames(pc[:n])
	st := stackTrace{}
	for {
		frame, more := frames.Next()
		st = append(st, stackFrame{
			fn:   frame.Function,
			file: frame.File,
			line: frame.Line,
		})
		if !more {
			break
		}
	}
	return st
}
