/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import coreutils "github.com/voedger/voedger/pkg/utils"

type OrFilter struct {
	filters []IFilter
}

func (f OrFilter) IsMatch(fd coreutils.FieldsDef, outputRow IOutputRow) (bool, error) {
	for _, filter := range f.filters {
		match, err := filter.IsMatch(fd, outputRow)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
