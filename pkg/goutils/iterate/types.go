/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iterate

// ForEachFunction is function type what enumerates all data in slice-like structures
type ForEachFunction[T any] func(enum func(T))

type ForEachFunction1Arg[T any, A1 any] func(arg1 A1, enum func(T))

type ForEachFunction2Value[V1 any, V2 any] func(enum func(V1, V2))

// ForEachMapFunc is function type what enumerates all key-value pairs in map-like structures
type ForEachMapFunc[K comparable, V any] func(enum func(K, V))
