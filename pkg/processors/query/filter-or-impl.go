/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import coreutils "github.com/untillpro/voedger/pkg/utils"

type OrFilter struct {
	filters []IFilter
}

func (f OrFilter) IsMatch(schemaFields coreutils.SchemaFields, outputRow IOutputRow) (bool, error) {
	for _, filter := range f.filters {
		match, err := filter.IsMatch(schemaFields, outputRow)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
