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
		int32Val, err := coreutils.ClarifyJSONNumber[int32](f.value.(json.Number))
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(int32) == int32Val, nil
	case appdef.DataKind_int64:
		int64Val, err := coreutils.ClarifyJSONNumber[int64](f.value.(json.Number))
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(int64) == int64Val, nil
	case appdef.DataKind_float32:
		float32Val, err := coreutils.ClarifyJSONNumber[float32](f.value.(json.Number))
		if err != nil {
			return false, err
		}
		return nearlyEqual(float64(float32Val), float64(outputRow.Value(f.field).(float32)), f.epsilon), nil
	case appdef.DataKind_float64:
		float64Val, err := coreutils.ClarifyJSONNumber[float64](f.value.(json.Number))
		if err != nil {
			return false, err
		}
		return nearlyEqual(float64Val, outputRow.Value(f.field).(float64), f.epsilon), nil
	case appdef.DataKind_string, appdef.DataKind_QName:
		return outputRow.Value(f.field).(string) == f.value.(string), nil
	case appdef.DataKind_bool:
		return outputRow.Value(f.field).(bool) == f.value.(bool), nil
	case appdef.DataKind_RecordID:
		recordID, err := coreutils.ClarifyJSONNumber[istructs.RecordID](f.value.(json.Number))
		if err != nil {
			return false, err
		}
		return outputRow.Value(f.field).(istructs.RecordID) == recordID, nil
	case appdef.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind_Eq, f.field, ErrWrongType)
	}
}
