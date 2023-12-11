/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=ProcKind -output=proc-kind_string.go

const (
	ProcKind_Command ProcKind = iota
	ProcKind_Query
	ProcKind_Projector

	ProcKind_Count
)

func (k ProcKind) MarshalText() ([]byte, error) {
	var s string
	if k < ProcKind_Count {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an ProcKind in human-readable form, without `ProcKind_` prefix,
// suitable for debugging or error messages
func (k ProcKind) TrimString() string {
	const pref = "ProcKind_"
	return strings.TrimPrefix(k.String(), pref)
}
