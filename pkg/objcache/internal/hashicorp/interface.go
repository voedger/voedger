/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package hashicorp

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

// LRU cache implemented by hashicorp LRU cache
type Cache[K comparable, V any] struct {
	lru *lru.Cache[K, V]
}

func new[K comparable, V any](size int, onEvicted func(K, V)) (c *Cache[K, V]) {
	var err error
	c = &Cache[K, V]{}
	if onEvicted != nil {
		c.lru, err = lru.NewWithEvict[K, V](size, onEvicted)
	} else {
		c.lru, err = lru.New[K, V](size)
	}
	if err != nil {
		panic(err)
	}

	return c
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	return c.lru.Get(key)
}

func (c *Cache[K, V]) Put(key K, value V) {
	_ = c.lru.Add(key, value)
}
