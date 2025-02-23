/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

type IVVMAppTTLStorage interface {
	InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error)
	CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error)
	CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error)
}

// ITTLStorage defines a TTL-based storage layer with explicit durations.
type ITTLStorage[K any, V any] interface {
	// InsertIfNotExist tries to insert (key, val) with a TTL only if key does not exist.
	// Returns (true, nil) if inserted successfully,
	// (false, nil) if the key already exists,
	// or (false, err) if a storage error occurs.
	InsertIfNotExist(key K, val V, ttlSeconds int) (bool, error)

	// CompareAndSwap checks if the current value for `key` is `oldVal`.
	// If it matches, sets it to `newVal` and updates the TTL to `ttl`.
	// Returns (true, nil) on success, (false, nil) if values do not match, or (false, err) on error.
	CompareAndSwap(key K, oldVal V, newVal V, ttlSeconds int) (bool, error)

	// CompareAndDelete compares the current value for `key` with `val`,
	// and if they match, deletes the key, returning (true, nil). Otherwise, (false, nil).
	// On storage error, returns (false, err).
	CompareAndDelete(key K, val V) (bool, error)
}
