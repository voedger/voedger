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
	ProjectorEventKind_Execute
	ProjectorEventKind_ExecuteWithParam

	ProjectorEventKind_Count
)

// Events for record any change.
var ProjectorEventKind_AnyChanges = []ProjectorEventKind{
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

// Returns is event kind compatible with type kind.
//
// # Compatibles:
//
//   - Any document or record can be inserted.
//   - Any document or record, except ODoc and ORecord, can be updated, activated or deactivated.
//   - Only command can be executed.
func (i ProjectorEventKind) typeCompatible(kind TypeKind) bool {
	switch i {
	case ProjectorEventKind_Insert, ProjectorEventKind_Update, ProjectorEventKind_Activate, ProjectorEventKind_Deactivate:
		return kind == TypeKind_GDoc || kind == TypeKind_GRecord ||
			kind == TypeKind_CDoc || kind == TypeKind_CRecord ||
			kind == TypeKind_WDoc || kind == TypeKind_WRecord
	case ProjectorEventKind_Execute:
		return kind == TypeKind_Command
	case ProjectorEventKind_ExecuteWithParam:
		return kind == TypeKind_Object || kind == TypeKind_ODoc
	}
	return false
}
