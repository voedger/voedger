/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package floatdrop

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	v := c.lru.Get(key)
	if v != nil {
		value = *v
		ok = true
	}
	return value, ok
}

func (c *Cache[K, V]) Put(key K, value V) {
	evicted := c.lru.Set(key, value)
	if c.onEvicted != nil {
		if evicted != nil {
			c.onEvicted(evicted.Key, evicted.Value)
		}
	}
}
