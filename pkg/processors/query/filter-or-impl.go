/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

type OrFilter struct {
	filters []IFilter
}

func (f OrFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	for _, filter := range f.filters {
		match, err := filter.IsMatch(fk, outputRow)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
