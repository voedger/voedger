/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package objcache

import "github.com/voedger/voedger/pkg/objcache/internal/hashicorp"

// Creates and return new LRU object cache with K key type and V value type.
//
// Maximum cache size is limited by size param. Optional onEvicted cb is called then some value evicted from cache.
func New[K comparable, V any](size int, onEvicted func(K, V)) ICache[K, V] {
	return hashicorp.New[K, V](size, onEvicted)
}
