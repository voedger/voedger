/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package objcache

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

// internally used interface
type refCounter interface {
	tryAddRef() bool
	Release()
}

// LRU cache implemented by hashicorp LRU cache
type cache[K comparable, V any] struct {
	lru *lru.Cache[K, V]
}

func (c *cache[K, V]) Get(key K) (value V, ok bool) {
	value, ok = c.lru.Get(key)
	if ok {
		if ref, auto := any(value).(refCounter); auto {
			ok = ref.tryAddRef()
		}
	}
	return value, ok
}

func (c *cache[K, V]) Put(key K, value V) {
	// The problem of the twice put (value won’t be freed) is solved by convention (*DO NOT PUT SAME VALUE TWICE*)
	// if old, exists := c.lru.Peek(key); exists && (any(old) == any(value)) { return }

	if ref, auto := any(value).(refCounter); auto {
		if !ref.tryAddRef() {
			// notest: looks like value right now released
			return
		}
	}
	_ = c.lru.Add(key, value)
}

func (c *cache[K, V]) evicted(_ K, value V) {
	if ref, ok := any(value).(refCounter); ok {
		ref.Release()
	}
}
