/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package imcache

import "github.com/erni27/imcache"

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	return c.cache.Get(key)
}

func (c *Cache[K, V]) Put(key K, value V) {
	c.cache.Set(key, value, imcache.WithNoExpiration())
}
