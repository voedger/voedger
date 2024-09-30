/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

type EqualsFilter struct {
	field   string
	value   interface{}
	epsilon float64
}

func (f EqualsFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	switch fk[f.field] {
	case appdef.DataKind_int32:
		int32Intf, err := coreutils.ClarifyJSONNumber(f.value.(json.Number), appdef.DataKind_int32)
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(int32) == int32Intf.(int32), nil
	case appdef.DataKind_int64:
		int64Intf, err := coreutils.ClarifyJSONNumber(f.value.(json.Number), appdef.DataKind_int64)
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(int64) == int64Intf.(int64), nil
	case appdef.DataKind_float32:
		float32Intf, err := coreutils.ClarifyJSONNumber(f.value.(json.Number), appdef.DataKind_float32)
		if err != nil {
			return false, err
		}
		return nearlyEqual(float64(float32Intf.(float32)), float64(outputRow.Value(f.field).(float32)), f.epsilon), nil
	case appdef.DataKind_float64:
		float64Intf, err := coreutils.ClarifyJSONNumber(f.value.(json.Number), appdef.DataKind_float64)
		if err != nil {
			return false, err
		}
		return nearlyEqual(float64Intf.(float64), outputRow.Value(f.field).(float64), f.epsilon), nil
	case appdef.DataKind_string:
		return outputRow.Value(f.field).(string) == f.value.(string), nil
	case appdef.DataKind_bool:
		return outputRow.Value(f.field).(bool) == f.value.(bool), nil
	case appdef.DataKind_RecordID:
		recordIDIntf, err := coreutils.ClarifyJSONNumber(f.value.(json.Number), appdef.DataKind_RecordID)
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(istructs.RecordID) == recordIDIntf.(istructs.RecordID), nil
	case appdef.DataKind_QName:
		return outputRow.Value(f.field).(string) == f.value.(string), nil
	case appdef.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind_Eq, f.field, ErrWrongType)
	}
}
