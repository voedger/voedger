/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import "github.com/voedger/voedger/pkg/exttinygo"

type QName = string

type Ref int64

type Bytes []byte

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
