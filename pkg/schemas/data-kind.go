/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=DataKind -output=data-kind_string.go

const (
	// null - no-value type. Returned when the requisted type does not exist
	DataKind_null DataKind = iota

	DataKind_int32
	DataKind_int64
	DataKind_float32
	DataKind_float64
	DataKind_bytes
	DataKind_string
	DataKind_QName
	DataKind_bool

	DataKind_RecordID

	// Complex types

	DataKind_Record
	DataKind_Event

	DataKind_FakeLast
)

// Returns is fixed width data kind
func (k DataKind) IsFixed() bool {
	switch k {
	case
		DataKind_int32,
		DataKind_int64,
		DataKind_float32,
		DataKind_float64,
		DataKind_QName,
		DataKind_bool,
		DataKind_RecordID:
		return true
	}
	return false
}

func (k DataKind) MarshalText() ([]byte, error) {
	var s string
	if k < DataKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an DataKind in human-readable form, without "DataKind_" prefix,
// suitable for debugging or error messages
func (k DataKind) ToString() string {
	const pref = "DataKind_"
	s := k.String()
	if strings.HasPrefix(s, pref) {
		s = s[len(pref):]
	}
	return s
}
