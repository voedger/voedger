/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"cmp"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func compareOrdered[T cmp.Ordered](a, b T, gt bool) bool {
	if gt {
		return a > b
	}
	return a < b
}

func matchOrdered(filterKind, field string, gt bool, fk FieldsKinds, outputRow IOutputRow, value interface{}) (bool, error) {
	switch fk[field] {
	case appdef.DataKind_int32:
		return compareOrdered(outputRow.Value(field).(int32), int32(value.(float64)), gt), nil
	case appdef.DataKind_int64:
		return compareOrdered(outputRow.Value(field).(int64), int64(value.(float64)), gt), nil
	case appdef.DataKind_float32:
		return compareOrdered(outputRow.Value(field).(float32), float32(value.(float64)), gt), nil
	case appdef.DataKind_float64:
		return compareOrdered(outputRow.Value(field).(float64), value.(float64), gt), nil
	case appdef.DataKind_string:
		return compareOrdered(outputRow.Value(field).(string), value.(string), gt), nil
	case appdef.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind, field, ErrWrongType)
	}
}
