/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

// panics on an unsupported kind guessing that pair <name, kind> could be taken from IDef.Fields() callback only
func ReadByKind(name appdef.FieldName, kind appdef.DataKind, rr istructs.IRowReader) interface{} {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("ReadByKind() failed: name %s, kind %s, source QName %s: %v", name, kind.String(), rr.AsQName(appdef.SystemField_QName), r))
			panic(r)
		}
	}()
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
	t := appDef.Type(qn)

	opts := &mapperOpts{}
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	proceedField := func(fieldName appdef.FieldName, kind appdef.DataKind) {
		if opts.filter != nil {
			if !opts.filter(fieldName, kind) {
				return
			}
		}
		if kind == appdef.DataKind_Record {
			if v, ok := obj.(istructs.IValue); ok {
				res[fieldName] = FieldsToMap(v.AsRecord(fieldName), appDef, optFuncs...)
			} else {
				panic("DataKind_Record field met -> IValue must be provided")
			}
		} else {
			res[fieldName] = ReadByKind(fieldName, kind, obj)
		}
	}

	if fields, ok := t.(appdef.IFields); ok {
		if opts.nonNilsOnly {
			for fieldName := range obj.FieldNames {
				proceedField(fieldName, fields.Field(fieldName).DataKind())
			}
		} else {
			for _, f := range fields.Fields() {
				proceedField(f.Name(), f.DataKind())
			}
		}
	}

	return res
}

func ObjectToMap(obj istructs.IObject, appDef appdef.IAppDef, opts ...MapperOpt) (res map[string]interface{}) {
	if obj.QName() == appdef.NullQName {
		return map[string]interface{}{}
	}
	res = FieldsToMap(obj, appDef, opts...)
	for container := range obj.Containers {
		cont := []map[string]interface{}{}
		for c := range obj.Children(container) {
			cont = append(cont, ObjectToMap(c, appDef, opts...))
		}
		res[container] = cont
	}
	return res
}

type cudsOpts struct {
	filter     func(appdef.QName) bool
	mapperOpts []MapperOpt
}

type CUDsOpt func(*cudsOpts)

func WithFilter(filterFunc func(appdef.QName) bool) CUDsOpt {
	return func(co *cudsOpts) {
		co.filter = filterFunc
	}
}

func WithMapperOpts(opts ...MapperOpt) CUDsOpt {
	return func(co *cudsOpts) {
		co.mapperOpts = opts
	}
}

func CUDsToMap(event istructs.IDbEvent, appDef appdef.IAppDef, optFuncs ...CUDsOpt) []map[string]interface{} {
	cuds := make([]map[string]interface{}, 0)
	opts := cudsOpts{}
	for _, f := range optFuncs {
		f(&opts)
	}
	for rec := range event.CUDs {
		if opts.filter != nil && !opts.filter(rec.QName()) {
			continue
		}
		cudData := make(map[string]interface{})
		cudData["sys.ID"] = rec.ID()
		cudData["sys.QName"] = rec.QName().String()
		cudData["IsNew"] = rec.IsNew()
		cudData["fields"] = FieldsToMap(rec, appDef, opts.mapperOpts...)
		cuds = append(cuds, cudData)
	}
	return cuds
}

func JSONMapToCUDBody(data []map[string]interface{}) string {
	cuds := make([]CUD, 0, len(data))
	for _, record := range data {
		c := CUD{
			Fields: make(map[string]interface{}),
		}
		for field, value := range record {
			c.Fields[field] = value
		}
		cuds = append(cuds, c)
	}
	bb, err := json.Marshal(CUDs{Cuds: cuds})
	if err != nil {
		panic(err)
	}
	return string(bb)
}

func CheckValueByKind(val interface{}, kind appdef.DataKind) error {
	ok := false
	switch val.(type) {
	case int32:
		ok = kind == appdef.DataKind_int32
	case int64:
		ok = kind == appdef.DataKind_int64 || kind == appdef.DataKind_RecordID
	case float32:
		ok = kind == appdef.DataKind_float32
	case float64:
		ok = kind == appdef.DataKind_float64
	case bool:
		ok = kind == appdef.DataKind_bool
	case string:
		ok = kind == appdef.DataKind_string || kind == appdef.DataKind_QName
	case []byte:
		ok = kind == appdef.DataKind_bytes
	}
	if !ok {
		return fmt.Errorf("provided value %v has type %T but %s is expected: %w", val, val, kind.String(), appdef.ErrInvalidError)
	}
	return nil
}
