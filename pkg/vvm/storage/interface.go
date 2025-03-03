/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */
package storage

// [~server.design.orch/ISysVvmStorage~impl]
type ISysVvmStorage interface {
	InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error)
	CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error)
	CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error)
	Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)
}
