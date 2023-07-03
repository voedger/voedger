/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package bytespool

import "sync"

var bp = sync.Pool{
	New: func() any { return []byte{} },
}

func Get() []byte { return bp.Get().([]byte) }

func Put(b []byte) {
	const maxCapacity = 64 * 1024
	if cap(b) < maxCapacity {
		b = (b)[:0]
		bp.Put(b)
	}
}
