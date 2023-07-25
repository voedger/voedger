/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

// internally used interface
type automated interface {
	AddRef() bool
	Release()
}

// LRU cache implemented by hashicorp LRU cache
type cache[K comparable, V any] struct {
	lru *lru.Cache[K, V]
}

func (c *cache[K, V]) Get(key K) (value V, ok bool) {
	value, ok = c.lru.Get(key)
	if ok {
		if ref, auto := any(value).(automated); auto {
			ok = ref.AddRef()
		}
	}
	return value, ok
}

func (c *cache[K, V]) Put(key K, value V) {
	if ref, auto := any(value).(automated); auto {
		if !ref.AddRef() {
			// notest: looks like value right now released
			return
		}
	}
	_ = c.lru.Add(key, value)
}

func (c *cache[K, V]) evicted(_ K, value V) {
	if ref, ok := any(value).(automated); ok {
		ref.Release()
	}
}
