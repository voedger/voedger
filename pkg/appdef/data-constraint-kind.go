/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=DataConstraintKind -output=data-constraint-kind_string.go

const (
	// null - no-value type. Returned when the requested kind does not exist
	DataConstraintKind_null DataConstraintKind = iota

	DataConstraintKind_MinLen
	DataConstraintKind_MaxLen
	DataConstraintKind_Pattern

	DataConstraintKind_Count
)

func (k DataConstraintKind) MarshalText() ([]byte, error) {
	var s string
	if k < DataConstraintKind_Count {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an DataConstraintKind in human-readable form, without "DataConstraintKind_" prefix,
// suitable for debugging or error messages
func (k DataConstraintKind) TrimString() string {
	const pref = "DataConstraintKind_"
	return strings.TrimPrefix(k.String(), pref)
}
