/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// panics on an unsupported kind guessing that pair <name, kind> could be taken from IDef.Fields() callback only
func ReadByKind(name string, kind appdef.DataKind, rr istructs.IRowReader) interface{} {
	switch kind {
	case appdef.DataKind_int32:
		return rr.AsInt32(name)
	case appdef.DataKind_int64:
		return rr.AsInt64(name)
	case appdef.DataKind_float32:
		return rr.AsFloat32(name)
	case appdef.DataKind_float64:
		return rr.AsFloat64(name)
	case appdef.DataKind_bytes:
		return rr.AsBytes(name)
	case appdef.DataKind_string:
		return rr.AsString(name)
	case appdef.DataKind_RecordID:
		return rr.AsRecordID(name)
	case appdef.DataKind_QName:
		return rr.AsQName(name).String()
	case appdef.DataKind_bool:
		return rr.AsBool(name)
	default:
		panic("unsupported kind " + fmt.Sprint(kind) + " for field " + name)
	}
}

type mapperOpts struct {
	filter      func(name string, kind appdef.DataKind) bool
	nonNilsOnly bool
}

type MapperOpt func(opt *mapperOpts)

func Filter(filterFunc func(name string, kind appdef.DataKind) bool) MapperOpt {
	return func(opts *mapperOpts) {
		opts.filter = filterFunc
	}
}

func WithNonNilsOnly() MapperOpt {
	return func(opts *mapperOpts) {
		opts.nonNilsOnly = true
	}
}

func FieldsToMap(obj istructs.IRowReader, appDef appdef.IAppDef, optFuncs ...MapperOpt) (res map[string]interface{}) {
	res = map[string]interface{}{}

	qn := obj.AsQName(appdef.SystemField_QName)
	if qn == appdef.NullQName {
		return
	}
	def := appDef.Def(qn)

	opts := &mapperOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	if opts.nonNilsOnly {
		obj.FieldNames(func(fieldName string) {
			kind := def.Field(fieldName).DataKind()
			if opts.filter != nil {
				if !opts.filter(fieldName, kind) {
					return
				}
			}
			if kind == appdef.DataKind_Record {
				if ival, ok := obj.(istructs.IValue); ok {
					res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), appDef, optFuncs...)
				} else {
					panic("DataKind_Record field met -> IValue must be provided")
				}
			} else {
				res[fieldName] = ReadByKind(fieldName, kind, obj)
			}
		})
	} else {
		def.Fields(
			func(f appdef.IField) {
				fieldName, kind := f.Name(), f.DataKind()
				if opts.filter != nil {
					if !opts.filter(fieldName, kind) {
						return
					}
				}
				if kind == appdef.DataKind_Record {
					if ival, ok := obj.(istructs.IValue); ok {
						res[fieldName] = FieldsToMap(ival.AsRecord(fieldName), appDef, optFuncs...)
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

func ObjectToMap(obj istructs.IObject, appDef appdef.IAppDef, opts ...MapperOpt) (res map[string]interface{}) {
	if obj.AsQName(appdef.SystemField_QName) == appdef.NullQName {
		return map[string]interface{}{}
	}
	res = FieldsToMap(obj, appDef, opts...)
	obj.Containers(func(container string) {
		var elemMap map[string]interface{}
		cont := []map[string]interface{}{}
		obj.Elements(container, func(el istructs.IElement) {
			elemMap = ObjectToMap(el, appDef, opts...)
			cont = append(cont, elemMap)
		})
		res[container] = cont
	})
	return res
}
