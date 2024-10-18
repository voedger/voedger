/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package objcache

import lru "github.com/hashicorp/golang-lru/v2"

// Creates and return new LRU object cache with K key type and V value type.
//
// Maximum cache size is limited by size param
func New[K comparable, V any](size int) ICache[K, V] {
	var err error
	c := &cache[K, V]{}
	c.lru, err = lru.NewWithEvict(size, c.evicted)
	if err != nil {
		// notest
		panic(err)
	}

	return c
}
