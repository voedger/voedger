/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iterate

// ForEachFunction is function type what enumerates all data in slice-like structures
type ForEachFunction[T any] func(enum func(T))

// ForEachMapFunc is function type what enumerates all key-value pairs in map-like structures
type ForEachMapFunc[K comparable, V any] func(enum func(K, V))
