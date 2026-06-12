/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

type GreaterFilter struct {
	field string
	value interface{}
}

func (f GreaterFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	return matchOrdered(filterKind_Gt, f.field, true, fk, outputRow, f.value)
}
