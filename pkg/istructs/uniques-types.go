/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istructs

type IUniques interface {
	GetAll(name QName) (uniques []IUnique)

	// fields order has no sense
	// only one unique could match. None matched -> nil
	GetForKeySet(qName QName, keyFieldsSet []string) IUnique
}

type IUnique interface {
	Fields() []string
	QName() QName
}
