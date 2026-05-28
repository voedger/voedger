/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

type LessFilter struct {
	field string
	value interface{}
}

func (f LessFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	return matchOrdered(filterKind_Lt, f.field, false, fk, outputRow, f.value)
}
