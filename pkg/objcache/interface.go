/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

// Objects cache
type ICache[K comparable, V any] interface {
	// Gets value by key. Returns true and value if key exists, false and nil overwise
	Get(K) (value V, ok bool)

	// Puts value with key
	Put(K, V)
}
