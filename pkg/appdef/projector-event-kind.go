/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author Maxim Geraskin
 */

package appdef

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=ProjectorEventKind -output=projector-event-kind_string.go

const (
	ProjectorEventKind_Insert ProjectorEventKind = iota + 1
	ProjectorEventKind_Update
	ProjectorEventKind_Activate
	ProjectorEventKind_Deactivate

	ProjectorEventKind_Count
)

var ProjectorEventKind_Any = []ProjectorEventKind{
	ProjectorEventKind_Insert,
	ProjectorEventKind_Update,
	ProjectorEventKind_Activate,
	ProjectorEventKind_Deactivate,
}

func (i ProjectorEventKind) MarshalText() ([]byte, error) {
	var s string
	if (i > 0) && (i < ProjectorEventKind_Count) {
		s = i.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(i), base)
	}
	return []byte(s), nil
}

// Renders an ProjectorEventKind in human-readable form, without `ProjectorEventKind_` prefix,
// suitable for debugging or error messages
func (i ProjectorEventKind) TrimString() string {
	const pref = "ProjectorEventKind_"
	return strings.TrimPrefix(i.String(), pref)
}
