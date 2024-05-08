/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// Renders an ProjectorEventKind in human-readable form, without `ProjectorEventKind_` prefix,
// suitable for debugging or error messages
func (i ProjectorEventKind) TrimString() string {
	const pref = "ProjectorEventKind_"
	return strings.TrimPrefix(i.String(), pref)
}

// Returns all projector event kinds what compatible with specified type.
//
// # Compatibles:
//
//   - Any Document or Record, except ODoc and ORecord, can be inserted, updated, activated or deactivated.
//   - Command can be executed.
//   - Object or ODoc can be parameter for command execute with.
func allProjectorEventsOnType(typ IType) set.Set[ProjectorEventKind] {
	var (
		recEvents = set.From(ProjectorEventKind_Insert, ProjectorEventKind_Update, ProjectorEventKind_Activate, ProjectorEventKind_Deactivate)

		anyKinds = map[QName]set.Set[ProjectorEventKind]{
			QNameAnyRecord:    recEvents,
			QNameAnyGDoc:      recEvents,
			QNameAnyCDoc:      recEvents,
			QNameAnyWDoc:      recEvents,
			QNameAnySingleton: recEvents,

			QNameAnyCommand: set.From(ProjectorEventKind_Execute),
			QNameAnyObject:  set.From(ProjectorEventKind_ExecuteWithParam),
			QNameAnyODoc:    set.From(ProjectorEventKind_ExecuteWithParam),
		}

		typeKinds = map[TypeKind]set.Set[ProjectorEventKind]{
			TypeKind_GDoc:    recEvents,
			TypeKind_GRecord: recEvents,
			TypeKind_CDoc:    recEvents,
			TypeKind_CRecord: recEvents,
			TypeKind_WDoc:    recEvents,
			TypeKind_WRecord: recEvents,

			TypeKind_Command: set.From(ProjectorEventKind_Execute),
			TypeKind_Object:  set.From(ProjectorEventKind_ExecuteWithParam),
			TypeKind_ODoc:    set.From(ProjectorEventKind_ExecuteWithParam),
		}
	)

	switch typ.Kind() {
	case TypeKind_Any:
		if kinds, ok := anyKinds[typ.QName()]; ok {
			return kinds
		}
	default:
		if kinds, ok := typeKinds[typ.Kind()]; ok {
			return kinds
		}
	}

	return set.Empty[ProjectorEventKind]()
}

// Returns is event kind compatible with type kind.
//
// # Compatibles:
//
//   - Any document or record, except ODoc and ORecord, can be inserted, updated, activated or deactivated.
//   - Only command can be executed.
//   - Only object or ODoc can be parameter for command execute with.
func projectorEventCompatableWith(event ProjectorEventKind, typ IType) (ok bool, err error) {
	if allProjectorEventsOnType(typ).Contains(event) {
		return true, nil
	}

	return false, ErrIncompatible("projector event «%v» is not compatible with %v", event, typ)
}
