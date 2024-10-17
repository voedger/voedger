/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iterate

// Slice is a function type wrapper for naked slices.
// Slice result can be passed as a first argument to `ForEach` routines
func Slice[T any](slice []T) ForEachFunction[T] {
	return func(enum func(T)) {
		for _, d := range slice {
			enum(d)
		}
	}
}

// Map is a function type wrapper for naked maps.
// Map result can be passed as a first argument to `FindFirstMap`
func Map[K comparable, V any](m map[K]V) ForEachMapFunc[K, V] {
	return func(enum func(K, V)) {
		for k, v := range m {
			enum(k, v)
		}
	}
}

// FindFirstMap find first key-value pair by `forEach` iterator, using test function.
func FindFirstMap[K comparable, V any](forEach ForEachMapFunc[K, V], test func(K, V) bool) (ok bool, key K, value V) {
	forEach(func(k K, v V) {
		if ok {
			return
		}
		if ok = test(k, v); ok {
			key = k
			value = v
		}
	})
	return ok, key, value
}
