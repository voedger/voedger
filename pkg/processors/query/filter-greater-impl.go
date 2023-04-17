/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type GreaterFilter struct {
	field string
	value interface{}
}

func (f GreaterFilter) IsMatch(schemaFields coreutils.SchemaFields, outputRow IOutputRow) (bool, error) {
	switch schemaFields[f.field] {
	case istructs.DataKind_int32:
		return outputRow.Value(f.field).(int32) > int32(f.value.(float64)), nil
	case istructs.DataKind_int64:
		return outputRow.Value(f.field).(int64) > int64(f.value.(float64)), nil
	case istructs.DataKind_float32:
		return outputRow.Value(f.field).(float32) > float32(f.value.(float64)), nil
	case istructs.DataKind_float64:
		return outputRow.Value(f.field).(float64) > f.value.(float64), nil
	case istructs.DataKind_string:
		return outputRow.Value(f.field).(string) > f.value.(string), nil
	case istructs.DataKind_null:
		return false, nil
	default:
		return false, fmt.Errorf("'%s' filter: field %s: %w", filterKind_Gt, f.field, ErrWrongType)
	}
}
