/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

import "github.com/voedger/voedger/pkg/schemas"

type IUniques interface {
	GetAll(name schemas.QName) (uniques []IUnique)

	// fields order has no sense
	// only one unique could match. None matched -> nil
	GetForKeySet(qName schemas.QName, keyFieldsSet []string) IUnique
}

type IUnique interface {
	Fields() []string
	QName() schemas.QName
}
