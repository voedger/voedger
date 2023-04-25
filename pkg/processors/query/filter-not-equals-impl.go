/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"

	"github.com/voedger/voedger/pkg/schemas"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type NotEqualsFilter struct {
	field   string
	value   interface{}
	epsilon float64
}

func (f NotEqualsFilter) IsMatch(schemaFields coreutils.SchemaFields, outputRow IOutputRow) (bool, error) {
	switch schemaFields[f.field] {
	case schemas.DataKind_int32:
		return outputRow.Value(f.field).(int32) != int32(f.value.(float64)), nil
	case schemas.DataKind_int64:
		return outputRow.Value(f.field).(int64) != int64(f.value.(float64)), nil
	case schemas.DataKind_float32:
		return !nearlyEqual(f.value.(float64), float64(outputRow.Value(f.field).(float32)), f.epsilon), nil
	case schemas.DataKind_float64:
		return !nearlyEqual(f.value.(float64), outputRow.Value(f.field).(float64), f.epsilon), nil
	case schemas.DataKind_string:
		return outputRow.Value(f.field).(string) != f.value.(string), nil
	case schemas.DataKind_bool:
		return outputRow.Value(f.field).(bool) != f.value.(bool), nil
	case schemas.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind_NotEq, f.field, ErrWrongType)
	}
}
