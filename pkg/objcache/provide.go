/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import lru "github.com/hashicorp/golang-lru/v2"

// Creates and return new LRU object cache with K key type and V value type.
//
// Maximum cache size is limited by size param. Optional onEvicted cb is called then some value evicted from cache.
func New[K comparable, V any](size int) ICache[K, V] {
	var err error
	c := &cache[K, V]{}
	c.lru, err = lru.NewWithEvict[K, V](size, c.evicted)
	if err != nil {
		// notest
		panic(err)
	}

	return c
}
