/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package pool

// IPool manages standalone and owned object lifetimes.
type IPool[T any] interface {
	// Get borrows an object with an initial reference count of one.
	Get() T

	// GetOwned borrows an object whose lifetime is managed by owner.
	// AddRef and Release panic on an owned object. Releasing owner
	// automatically releases the object. Concurrent calls with the same
	// owner are supported. GetOwned panics if owner is already released.
	GetOwned(owner IReleaser) T
}

// IReleaser manages references and returns its containing instance to the
// pool. A NewPool instantiator must store the provided IReleaser in the
// instance it creates.
type IReleaser interface {
	// AddRef adds a reference to the instance. Each reference must be
	// balanced by a Release call.
	// Panics if the instance is owned or has already been released.
	AddRef()

	// Release removes a reference from the instance. When the last reference
	// is removed, Release calls Cleanup if it exists and returns the instance
	// to the pool. It panics if the instance is owned or already released.
	Release()

	// IsOwned reports whether the instance was borrowed with GetOwned.
	IsOwned() bool

	// for internal use
	releaseOwned()
	reset()
	setIsOwned()
	setBorrowStackTrace(stackTrace string)
	init(obj interface{})
	beginOwnedBorrow() bool
	finishOwnedBorrow()
	addOwned(IReleaser)
	waitOwnedBorrows()
	setOwnedTail(IReleaser)
	takeOwnedTail() IReleaser
}
