/*
 * Copyright (c) 2025-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"context"
	"errors"
	"sort"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func readView(ctx context.Context, appDef appdef.IAppDef, viewRecords istructs.IViewRecords, params QueryParams, wsid istructs.WSID, view appdef.QName) (objects []map[string]interface{}, err error) {
	objects, err = read(ctx, appDef, viewRecords, params, wsid, view)
	if err != nil {
		return
	}
	err = order(objects, params)
	if err != nil {
		return
	}
	applyKeys(objects, params)
	objects = subList(objects, params)
	return
}
func read(ctx context.Context, appDef appdef.IAppDef, viewRecords istructs.IViewRecords, params QueryParams, wsid istructs.WSID, view appdef.QName) (objects []map[string]interface{}, err error) {
	kb := viewRecords.KeyBuilder(view)
	kb.PutFromJSON(params.Constraints.Where)
	objects = make([]map[string]interface{}, 0)
	err = viewRecords.Read(ctx, wsid, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		object := coreutils.FieldsToMap(key, appDef, coreutils.WithNonNilsOnly())
		for k, v := range coreutils.FieldsToMap(value, appDef, coreutils.WithNonNilsOnly()) {
			object[k] = v
		}
		objects = append(objects, object)
		return
	})
	return
}
func order(objects []map[string]interface{}, params QueryParams) (err error) {
	if len(params.Constraints.Order) == 0 {
		return
	}
	sort.Slice(objects, func(i, j int) bool {
		for _, orderBy := range params.Constraints.Order {
			vi := objects[i][orderBy]
			vj := objects[j][orderBy]
			switch vi.(type) {
			case int32:
				return compare(vi.(int32), vj.(int32))
			case int64:
				return compare(vi.(int64), vj.(int64))
			case float32:
				return compare(vi.(float32), vj.(float32))
			case float64:
				return compare(vi.(float64), vj.(float64))
			case []byte:
				return compare(string(vi.([]byte)), string(vi.([]byte)))
			case string:
				return compare(vi.(string), vj.(string))
			case appdef.QName:
				return compare(vi.(appdef.QName).String(), vj.(appdef.QName).String())
			case bool:
				return vi.(bool) != vj.(bool)
			case istructs.RecordID:
				return compare(uint64(vi.(istructs.RecordID)), uint64(vj.(istructs.RecordID)))
			default:
				err = errors.New("unsupported type")
			}
		}
		return false
	})
	return
}
func applyKeys(objects []map[string]interface{}, params QueryParams) {
	if len(params.Constraints.Keys) == 0 {
		return
	}
	keys := make(map[string]bool)
	for _, key := range params.Constraints.Keys {
		keys[key] = true
	}
	for i := range objects {
		for k := range objects[i] {
			if keys[k] {
				continue
			}
			delete(objects[i], k)
		}
	}
}
func subList(objects []map[string]interface{}, params QueryParams) []map[string]interface{} {
	if params.Constraints.Limit == 0 && params.Constraints.Skip == 0 {
		return objects
	}
	if params.Constraints.Skip > len(objects) {
		return []map[string]interface{}{}
	}
	if params.Constraints.Skip+params.Constraints.Limit > len(objects) {
		return objects[params.Constraints.Skip:]
	}
	return objects[params.Constraints.Skip:params.Constraints.Limit]
}
func compare[E int32 | int64 | float32 | float64 | string | uint64](v1, v2 E) bool {
	return v1 < v2
}
