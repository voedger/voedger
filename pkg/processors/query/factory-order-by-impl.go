/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"fmt"

	"github.com/voedger/voedger/pkg/coreutils"
)

func NewOrderBy(data coreutils.MapObject) (IOrderBy, error) {
	field, err := data.AsStringRequired("field")
	if err != nil {
		return nil, fmt.Errorf("orderBy: %w", err)
	}
	desc, _, err := data.AsBoolean("desc")
	if err != nil {
		return nil, fmt.Errorf("orderBy: %w", err)
	}
	return orderBy{
		field: field,
		desc:  desc,
	}, nil
}

type orderBy struct {
	field string
	desc  bool
}

func (o orderBy) Field() string { return o.field }
func (o orderBy) IsDesc() bool  { return o.desc }
