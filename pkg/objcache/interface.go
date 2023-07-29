/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package objcache

// Objects cache
type ICache[K comparable, V any] interface {
	// Gets value by key. Returns true and value if key exists, false and
	// nil overwise
	//
	// If value has RefCounter, then
	//  - increments value reference counter
	//  - client should call value Release() after using value
	Get(K) (value V, ok bool)

	// Puts value into cache.
	//
	// If value has RefCounter, then
	//  - increments value reference counter
	//  - then the value will be evicted from the cache, value Release() will be called
	Put(K, V)
}
