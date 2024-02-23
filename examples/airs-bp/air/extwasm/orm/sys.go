/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

type QName = string

type ID int64

type Type struct {
	qname QName
}

func (t *Type) QName() QName {
	return t.qname
}

type Event struct {}
