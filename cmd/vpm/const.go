/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

const (
	pkgDirName           = "pkg"
	ormDirName           = "orm"
	defaultPermissions   = 0766
	baselineInfoFileName = "baseline.json"
	timestampFormat      = "Mon, 02 Jan 2006 15:04:05.000 GMT"
)

const (
	sysContent = `/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import exttinygo "github.com/voedger/exttinygo"

type QName = string

type Ref int64

func (r Ref) ID() ID { return ID(r) }

type ID int64

const (
	FieldNameEventArgumentObject = "ArgumentObject"
	FieldNameSysID               = "sys.ID"
)

type Type struct {
	qname QName
}

func (t *Type) QName() QName {
	return t.qname
}

type Event struct{}

type Value_CommandContext struct{ tv exttinygo.TValue }

func CommandContext() Value_CommandContext {
	kb := exttinygo.KeyBuilder(exttinygo.StorageCommandContext, exttinygo.NullEntity)
	return Value_CommandContext{tv: exttinygo.MustGetValue(kb)}
}
`
)
