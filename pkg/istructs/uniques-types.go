/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import "github.com/voedger/voedger/pkg/appdef"

// Deprecated: use IDef().Uniques
type IUniques interface {
	GetAll(name appdef.QName) (uniques []IUnique)
}

// Deprecated: use IDef().Uniques
type IUnique interface {
	Fields() []string
	QName() appdef.QName
}
