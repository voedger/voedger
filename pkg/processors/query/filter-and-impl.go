/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

type AndFilter struct {
	filters []IFilter
}

func (f AndFilter) IsMatch(fk FieldsKinds, outputRow IOutputRow) (bool, error) {
	for _, filter := range f.filters {
		match, err := filter.IsMatch(fk, outputRow)
		if err != nil {
			return false, err
		}
		if !match {
			return false, err
		}
	}
	return true, nil
}
