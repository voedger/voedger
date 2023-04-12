/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"fmt"

	"github.com/untillpro/voedger/pkg/istructs"
)

type SchemaFields map[string]istructs.DataKindType

func Read(fieldName string, sf SchemaFields, rr istructs.IRowReader) (val interface{}) {
	return ReadByKind(fieldName, sf[fieldName], rr)
}

func ReadValue(fieldName string, sf SchemaFields, schemas istructs.ISchemas, val istructs.IValue) (res interface{}) {
	if sf[fieldName] == istructs.DataKind_Record {
		return FieldsToMap(val.AsRecord(fieldName), schemas)
	}
	return ReadByKind(fieldName, sf[fieldName], val)
}

// panics on an unsupported kind guessing that pair <name, kind> could be taken from ISchema.Fields() callback only
func ReadByKind(name string, kind istructs.DataKindType, rr istructs.IRowReader) interface{} {
	switch kind {
	case istructs.DataKind_int32:
		return rr.AsInt32(name)
	case istructs.DataKind_int64:
		return rr.AsInt64(name)
	case istructs.DataKind_float32:
		return rr.AsFloat32(name)
	case istructs.DataKind_float64:
		return rr.AsFloat64(name)
	case istructs.DataKind_bytes:
		return rr.AsBytes(name)
	case istructs.DataKind_string:
		return rr.AsString(name)
	case istructs.DataKind_RecordID:
		return rr.AsRecordID(name)
	case istructs.DataKind_QName:
		return rr.AsQName(name).String()
	case istructs.DataKind_bool:
		return rr.AsBool(name)
	default:
		panic("unsupported kind " + fmt.Sprint(kind) + " for field " + name)
	}
}

func NewSchemaFields(schema istructs.ISchema) SchemaFields {
	fields := make(map[string]istructs.DataKindType)
	schema.Fields(func(fieldName string, kind istructs.DataKindType) {
		fields[fieldName] = kind
	})
	return fields
}

type mapperOpts struct {
	filter      func(name string, kind istructs.DataKindType) bool
	nonNilsOnly bool
}

type MapperOpt func(opt *mapperOpts)

func Filter(filterFunc func(name string, kind istructs.DataKindType) bool) MapperOpt {
	return func(opts *mapperOpts) {
		opts.filter = filterFunc
	}
}

func WithNonNilsOnly() MapperOpt {
	return func(opts *mapperOpts) {
		opts.nonNilsOnly = true
	}
}

func FieldsToMap(obj istructs.IRowReader, schemas istructs.ISchemas, optFuncs ...MapperOpt) (res map[string]interface{}) {
	res = map[string]interface{}{}
	if obj.AsQName(istructs.SystemField_QName) == istructs.NullQName {
		return
	}
	opts := &mapperOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	if opts.nonNilsOnly {
		s := schemas.Schema(obj.AsQName(istructs.SystemField_QName))
		sf := NewSchemaFields(s)
		obj.FieldNames(func(fieldName string) {
			kind := sf[fieldName]
			if opts.filter != nil {
				if !opts.filter(fieldName, kind) {
					return
				}
			}
			if kind == istructs.DataKind_Record {
				if ival, ok := obj.(istructs.IValue); ok {
					res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), schemas, optFuncs...)
				} else {
					panic("DataKind_Record field met -> IValue must be provided")
				}
			} else {
				res[fieldName] = ReadByKind(fieldName, kind, obj)
			}
		})
	} else {
		schemas.Schema(obj.AsQName(istructs.SystemField_QName)).Fields(func(fieldName string, kind istructs.DataKindType) {
			if opts.filter != nil {
				if !opts.filter(fieldName, kind) {
					return
				}
			}
			if kind == istructs.DataKind_Record {
				if ival, ok := obj.(istructs.IValue); ok {
					res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), schemas, optFuncs...)
				} else {
					panic("DataKind_Record field met -> IValue must be provided")
				}
			} else {
				res[fieldName] = ReadByKind(fieldName, kind, obj)
			}
		})
	}
	return res
}

func ObjectToMap(obj istructs.IObject, schemas istructs.ISchemas, opts ...MapperOpt) (res map[string]interface{}) {
	if obj.AsQName(istructs.SystemField_QName) == istructs.NullQName {
		return map[string]interface{}{}
	}
	res = FieldsToMap(obj, schemas, opts...)
	obj.Containers(func(container string) {
		var elemMap map[string]interface{}
		cont := []map[string]interface{}{}
		obj.Elements(container, func(el istructs.IElement) {
			elemMap = ObjectToMap(el, schemas, opts...)
			cont = append(cont, elemMap)
		})
		res[container] = cont
	})
	return res
}
