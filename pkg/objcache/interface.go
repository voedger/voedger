/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

// Objects cache
type ICache[K comparable, V any] interface {
	// Gets value by key. Returns true and value if key exists, false and
	// nil overwise
	//
	// If value supports IReleasable case, then
	//  - calls value AddRef()
	//  - client should call FreeRef() after using value
	Get(K) (value V, ok bool)

	// Puts value into cache.
	//
	// If value supports IReleasable case, then
	//  - calls value AddRef()
	//  - then the value will be evicted from the cache, FreeRef() will be called
	Put(K, V)
}
