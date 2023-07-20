/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package hashicorp

func New[K comparable, V any](size int, onEvict func(K, V)) *Cache[K, V] {
	return new[K, V](size, onEvict)
}
