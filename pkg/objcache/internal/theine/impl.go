/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package theine

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	return c.c.Get(key)
}

func (c *Cache[K, V]) Put(key K, value V) {
	_ = c.c.Set(key, value, 1)
}
