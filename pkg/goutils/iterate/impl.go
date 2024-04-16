/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package iterate

// ForEach pass `enum` callback to `forEach` iterator to call `enum` for each data from slice-like structures
func ForEach[T any](forEach ForEachFunction[T], enum func(T)) {
	forEach(enum)
}

func ForEachError2Values[V1 any, V2 any](forEach ForEachFunction2Value[V1, V2], do func(V1, V2) error) (err error) {
	forEach(func(v1 V1, v2 V2) {
		if err != nil {
			return
		}
		err = do(v1, v2)
	})
	return err
}

// Same as ForEachError but with an one addition arg
func ForEachError1Arg[T any, A1 any](forEach ForEachFunction1Arg[T, A1], arg1 A1, do func(T) error) (err error) {
	forEach(arg1, func(d T) {
		if err != nil {
			return
		}
		err = do(d)
	})
	return err
}

func ForEachError[T any](forEach ForEachFunction[T], do func(T) error) (err error) {
	forEach(func(d T) {
		if err != nil {
			return
		}
		err = do(d)
	})
	return err
}

// Slice is a function type wrapper for naked slices.
// Slice result can be passed as a first argument to `ForEach`, `FindFirst` and `FindFirstError` routines
func Slice[T any](slice []T) ForEachFunction[T] {
	return func(enum func(T)) {
		for _, d := range slice {
			enum(d)
		}
	}
}

// FindFirst find first data by `forEach` iterator, using test function.
func FindFirst[T any](forEach ForEachFunction[T], test func(T) bool) (ok bool, data T) {
	forEach(func(d T) {
		if ok {
			return
		}
		if ok = test(d); ok {
			data = d
		}
	})
	return ok, data
}

// FindFirstData find first occurs of specified `data` by `forEach` iterator
func FindFirstData[T comparable](forEach ForEachFunction[T], data T) (ok bool, idx int) {
	idx = -1
	i := 0
	forEach(func(d T) {
		if ok {
			return
		}
		if ok = (d == data); ok {
			idx = i
		}
		i++
	})
	return ok, idx
}

// FindFirstError find first data with error by `forEach` iterator, using test function.
func FindFirstError[T any](forEach ForEachFunction[T], test func(T) error) (data T, err error) {
	forEach(func(d T) {
		if err != nil {
			return
		}
		if err = test(d); err != nil {
			data = d
		}
	})
	return data, err
}

// ForEachMap pass `enum` callback to `forEach` iterator to call `enum` for each key-data pairs from map-like structures
func ForEachMap[K comparable, V any](forEach ForEachMapFunc[K, V], enum func(K, V)) {
	forEach(enum)
}

// Map is a function type wrapper for naked maps.
// Map result can be passed as a first argument to `ForEachMap`, `FindFirstMap` and `FindFirstMapError` routines
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

// FindFirstMapKey find first key-value pair by `forEach` iterator, using specified `key` value.
func FindFirstMapKey[K comparable, V any](forEach ForEachMapFunc[K, V], key K) (ok bool, foundedKey K, value V) {
	forEach(func(k K, v V) {
		if ok {
			return
		}
		if ok = (k == key); ok {
			foundedKey = k
			value = v
		}
	})
	return ok, foundedKey, value
}

// FindFirstError find first key-value pair with error by `forEach` iterator, using test function.
func FindFirstMapError[K comparable, V any](forEach ForEachMapFunc[K, V], test func(K, V) error) (key K, value V, err error) {
	forEach(func(k K, v V) {
		if err != nil {
			return
		}
		if err = test(k, v); err != nil {
			key = k
			value = v
		}
	})
	return key, value, err
}
