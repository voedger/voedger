/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package hashicorp

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	return c.lru.Get(key)
}

func (c *Cache[K, V]) Put(key K, value V) {
	_ = c.lru.Add(key, value)
}
