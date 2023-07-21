/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package floatdrop

import (
	lru "github.com/floatdrop/lru"
)

// LRU cache implemented by floatdrop LRU cache
type Cache[K comparable, V any] struct {
	lru       *lru.LRU[K, V]
	onEvicted func(K, V)
}

func new[K comparable, V any](size int, onEvicted func(K, V)) (c *Cache[K, V]) {
	c = &Cache[K, V]{
		lru:       lru.New[K, V](size),
		onEvicted: onEvicted,
	}

	return c
}
