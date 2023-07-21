/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package imcache

import (
	"github.com/erni27/imcache"
)

// LRU cache implemented by hashicorp LRU cache
type Cache[K comparable, V any] struct {
	cache *imcache.Cache[K, V]
}

func new[K comparable, V any](size int, onEvicted func(K, V)) (c *Cache[K, V]) {
	c = &Cache[K, V]{}
	if onEvicted != nil {
		c.cache = imcache.New[K, V](
			imcache.WithMaxEntriesOption[K, V](size),
			imcache.WithEvictionCallbackOption[K, V](
				func(k K, v V, _ imcache.EvictionReason) {
					onEvicted(k, v)
				}),
		)
	} else {
		c.cache = imcache.New[K, V](
			imcache.WithMaxEntriesOption[K, V](size),
		)
	}

	return c
}
