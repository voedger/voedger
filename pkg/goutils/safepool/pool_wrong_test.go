/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

//nolint:testableexamples // This example documents nondeterministic sync.Pool reuse.
package safepool

import (
	"fmt"
	"sync"

	"github.com/valyala/bytebufferpool"
)

// example struct to be pooled
type pooled_wrong struct {
	b *bytebufferpool.ByteBuffer
}

var pool = sync.Pool{
	New: func() interface{} {
		return &pooled_wrong{b: nil}
	},
}

func (p *pooled_wrong) Release() {
	// Returning the same pointer twice may let two borrowers receive it.
	// A release guard must reject the second call before returning either
	// the buffer or the object. Each manually pooled type needs this guard.
	bytebufferpool.Put(p.b)
	pool.Put(p)
}

func GetPooledStruct() *pooled_wrong {
	// problem: how to track borrowed but not returned?
	// see Example_doubleRelease()
	res := pool.Get().(*pooled_wrong)
	res.b = bytebufferpool.Get()
	return res
}

// Example_doubleRelease illustrates a missing release guard in a manual pool.
// It has no Output assertion because sync.Pool reuse is not guaranteed.
// Go compiles this example but does not run it without an Output section.
func Example_doubleRelease() {
	wrong := GetPooledStruct()
	wrong.Release()
	// This mistakenly returns both the same object and its buffer a second
	// time. Neither pool rejects the duplicate return.
	wrong.Release()

	new1 := GetPooledStruct()
	new2 := GetPooledStruct()

	// These two borrowers may now share the same object and buffer. Changes
	// made by one borrower would then affect the other unexpectedly.
	// sync.Pool may discard entries at any time, including in race-enabled
	// builds, so false here does not mean the duplicate release was safe.
	fmt.Println("same object borrowed twice:", new1 == new2)
}
