/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import coreutils "github.com/voedger/voedger/pkg/utils"

type AndFilter struct {
	filters []IFilter
}

func (f AndFilter) IsMatch(fd coreutils.FieldsDef, outputRow IOutputRow) (bool, error) {
	for _, filter := range f.filters {
		match, err := filter.IsMatch(fd, outputRow)
		if err != nil {
			return false, err
		}
		if !match {
			return false, err
		}
	}
	return true, nil
}
