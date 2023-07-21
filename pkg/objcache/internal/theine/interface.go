/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package theine

import (
	theine "github.com/Yiling-J/theine-go"
)

// LRU cache implemented by theine-go hybrid cache
type Cache[K comparable, V any] struct {
	c *theine.Cache[K, V]
}

func new[K comparable, V any](size int, onEvicted func(K, V)) (c *Cache[K, V]) {
	c = &Cache[K, V]{}
	bld := theine.NewBuilder[K, V](int64(size))
	if onEvicted != nil {
		bld.RemovalListener(func(key K, value V, _ theine.RemoveReason) { onEvicted(key, value) })
	}

	var err error

	if c.c, err = bld.Build(); err != nil {
		panic(err)
	}
	return c
}
