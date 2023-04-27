/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import "github.com/voedger/voedger/pkg/appdef"

type IUniques interface {
	GetAll(name appdef.QName) (uniques []IUnique)

	// fields order has no sense
	// only one unique could match. None matched -> nil
	GetForKeySet(qName appdef.QName, keyFieldsSet []string) IUnique
}

type IUnique interface {
	Fields() []string
	QName() appdef.QName
}
