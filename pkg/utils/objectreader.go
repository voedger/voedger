/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

type SchemaFields map[string]schemas.DataKind

func Read(fieldName string, sf SchemaFields, rr istructs.IRowReader) (val interface{}) {
	return ReadByKind(fieldName, sf[fieldName], rr)
}

func ReadValue(fieldName string, sf SchemaFields, schemaCache schemas.SchemaCache, val istructs.IValue) (res interface{}) {
	if sf[fieldName] == schemas.DataKind_Record {
		return FieldsToMap(val.AsRecord(fieldName), schemaCache)
	}
	return ReadByKind(fieldName, sf[fieldName], val)
}

// panics on an unsupported kind guessing that pair <name, kind> could be taken from ISchema.Fields() callback only
func ReadByKind(name string, kind schemas.DataKind, rr istructs.IRowReader) interface{} {
	switch kind {
	case schemas.DataKind_int32:
		return rr.AsInt32(name)
	case schemas.DataKind_int64:
		return rr.AsInt64(name)
	case schemas.DataKind_float32:
		return rr.AsFloat32(name)
	case schemas.DataKind_float64:
		return rr.AsFloat64(name)
	case schemas.DataKind_bytes:
		return rr.AsBytes(name)
	case schemas.DataKind_string:
		return rr.AsString(name)
	case schemas.DataKind_RecordID:
		return rr.AsRecordID(name)
	case schemas.DataKind_QName:
		return rr.AsQName(name).String()
	case schemas.DataKind_bool:
		return rr.AsBool(name)
	default:
		panic("unsupported kind " + fmt.Sprint(kind) + " for field " + name)
	}
}

func NewSchemaFields(schema schemas.Schema) SchemaFields {
	fields := make(map[string]schemas.DataKind)
	schema.Fields(
		func(f schemas.Field) {
			fields[f.Name()] = f.DataKind()
		})
	return fields
}

type mapperOpts struct {
	filter      func(name string, kind schemas.DataKind) bool
	nonNilsOnly bool
}

type MapperOpt func(opt *mapperOpts)

func Filter(filterFunc func(name string, kind schemas.DataKind) bool) MapperOpt {
	return func(opts *mapperOpts) {
		opts.filter = filterFunc
	}
}

func WithNonNilsOnly() MapperOpt {
	return func(opts *mapperOpts) {
		opts.nonNilsOnly = true
	}
}

func FieldsToMap(obj istructs.IRowReader, schemaCache schemas.SchemaCache, optFuncs ...MapperOpt) (res map[string]interface{}) {
	res = map[string]interface{}{}
	if obj.AsQName(schemas.SystemField_QName) == schemas.NullQName {
		return
	}
	opts := &mapperOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	if opts.nonNilsOnly {
		s := schemaCache.Schema(obj.AsQName(schemas.SystemField_QName))
		sf := NewSchemaFields(s)
		obj.FieldNames(func(fieldName string) {
			kind := sf[fieldName]
			if opts.filter != nil {
				if !opts.filter(fieldName, kind) {
					return
				}
			}
			if kind == schemas.DataKind_Record {
				if ival, ok := obj.(istructs.IValue); ok {
					res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), schemaCache, optFuncs...)
				} else {
					panic("DataKind_Record field met -> IValue must be provided")
				}
			} else {
				res[fieldName] = ReadByKind(fieldName, kind, obj)
			}
		})
	} else {
		schemaCache.Schema(obj.AsQName(schemas.SystemField_QName)).Fields(
			func(f schemas.Field) {
				fieldName, kind := f.Name(), f.DataKind()
				if opts.filter != nil {
					if !opts.filter(fieldName, kind) {
						return
					}
				}
				if kind == schemas.DataKind_Record {
					if ival, ok := obj.(istructs.IValue); ok {
						res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), schemaCache, optFuncs...)
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

func ObjectToMap(obj istructs.IObject, schemaCache schemas.SchemaCache, opts ...MapperOpt) (res map[string]interface{}) {
	if obj.AsQName(schemas.SystemField_QName) == schemas.NullQName {
		return map[string]interface{}{}
	}
	res = FieldsToMap(obj, schemaCache, opts...)
	obj.Containers(func(container string) {
		var elemMap map[string]interface{}
		cont := []map[string]interface{}{}
		obj.Elements(container, func(el istructs.IElement) {
			elemMap = ObjectToMap(el, schemaCache, opts...)
			cont = append(cont, elemMap)
		})
		res[container] = cont
	})
	return res
}
