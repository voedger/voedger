/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Empty name
const NullName = ""

// Null (empty) QName
var (
	QNameForNull  = NewQName(NullName, NullName)
	NullQName     = QNameForNull
	NullFullQName = NewFullQName(NullName, NullName)
)

// NullAppQName is undefined (or empty) application name
var NullAppQName = NewAppQName(NullName, NullName)

// NullType is used for return then type is not founded
var NullType = new(nullType)

// NullFields is used for return then IFields is not supported
var NullFields = new(nullFields)
