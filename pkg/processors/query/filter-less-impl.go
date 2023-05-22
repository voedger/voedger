/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

type LessFilter struct {
	field string
	value interface{}
}

func (f LessFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	switch fk[f.field] {
	case appdef.DataKind_int32:
		return outputRow.Value(f.field).(int32) < int32(f.value.(float64)), nil
	case appdef.DataKind_int64:
		return outputRow.Value(f.field).(int64) < int64(f.value.(float64)), nil
	case appdef.DataKind_float32:
		return outputRow.Value(f.field).(float32) < float32(f.value.(float64)), nil
	case appdef.DataKind_float64:
		return outputRow.Value(f.field).(float64) < f.value.(float64), nil
	case appdef.DataKind_string:
		return outputRow.Value(f.field).(string) < f.value.(string), nil
	case appdef.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind_Lt, f.field, ErrWrongType)
	}
}
