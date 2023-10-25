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

//go:generate stringer -type=ConstraintKind -output=constraint-kind_string.go

const (
	// null - no-value type. Returned when the requested kind does not exist
	ConstraintKind_null ConstraintKind = iota

	ConstraintKind_MinLen
	ConstraintKind_MaxLen
	ConstraintKind_Pattern

	ConstraintKind_MinIncl
	ConstraintKind_MinExcl
	ConstraintKind_MaxIncl
	ConstraintKind_MaxExcl

	ConstraintKind_Count
)

func (k ConstraintKind) MarshalText() ([]byte, error) {
	var s string
	if k < ConstraintKind_Count {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an DataConstraintKind in human-readable form, without "DataConstraintKind_" prefix,
// suitable for debugging or error messages
func (k ConstraintKind) TrimString() string {
	const pref = "ConstraintKind_"
	return strings.TrimPrefix(k.String(), pref)
}
