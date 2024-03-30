/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Michael Saigachenko
 */

package safestate

import "github.com/voedger/voedger/pkg/appdef"

type TSafeKeyBuilder int64

type ISafeState interface {
	KeyBuilder(storage, entity appdef.QName) TSafeKeyBuilder

	// CanExist(key TSafeKeyBuilder) (value IStateValue, ok bool, err error)

	// CanExistAll(keys []IStateKeyBuilder, callback StateValueCallback) (err error)

	// MustExist(key IStateKeyBuilder) (value IStateValue, err error)

	// MustExistAll(keys []IStateKeyBuilder, callback StateValueCallback) (err error)

	// MustNotExist(key IStateKeyBuilder) (err error)

	// MustNotExistAll(keys []IStateKeyBuilder) (err error)
}
