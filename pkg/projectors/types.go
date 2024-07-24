/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"slices"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/iterate"
	"github.com/voedger/voedger/pkg/istructs"
)

type asyncActualizerContextState struct {
	lock   sync.Mutex
	err    error
	ctx    context.Context
	cancel func()
}

func (s *asyncActualizerContextState) cancelWithError(err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.err = err
	s.cancel()
}

// @ConcurrentAccess
func (s *asyncActualizerContextState) error() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.err
}

func isAcceptable(event istructs.IPLogEvent, wantErrors bool, triggeringQNames map[appdef.QName][]appdef.ProjectorEventKind, appDef appdef.IAppDef) bool {
	switch event.QName() {
	case istructs.QNameForError:
		return wantErrors
	case istructs.QNameForCorruptedData:
		return false
	}

	if len(triggeringQNames) == 0 {
		return true
	}

	if triggeringKinds, ok := triggeringQNames[event.QName()]; ok {
		if slices.Contains(triggeringKinds, appdef.ProjectorEventKind_Execute) {
			return true
		}
	}

	if triggeringKinds, ok := triggeringQNames[event.ArgumentObject().QName()]; ok {
		// ON (some doc of kind ODoc)
		if slices.Contains(triggeringKinds, appdef.ProjectorEventKind_ExecuteWithParam) {
			return true
		}
	} else {
		// ON (ODoc)
		argumentTypeKind := appDef.Type(event.ArgumentObject().QName()).Kind()
		if argumentTypeKind == appdef.TypeKind_ODoc {
			if triggeringKinds, ok := triggeringQNames[istructs.QNameODoc]; ok {
				if slices.Contains(triggeringKinds, appdef.ProjectorEventKind_ExecuteWithParam) {
					return true
				}
			}
		}
	}

	triggered, _ := iterate.FindFirst(event.CUDs, func(rec istructs.ICUDRow) bool {
		triggeringKinds, ok := triggeringQNames[rec.QName()]
		if !ok {
			recType := appDef.Type(rec.QName())
			globalQNames := cudTypeKindToGlobalDocQNames[recType.Kind()]
			for _, globalQName := range globalQNames {
				if triggeringKinds, ok = triggeringQNames[globalQName]; ok {
					break
				}
			}
		}
		for _, triggerkingKind := range triggeringKinds {
			switch triggerkingKind {
			case appdef.ProjectorEventKind_Insert:
				if rec.IsNew() {
					return true
				}
			case appdef.ProjectorEventKind_Update:
				if !rec.IsNew() {
					return true
				}
			case appdef.ProjectorEventKind_Activate:
				if !rec.IsNew() {
					activated, _, _ := iterate.FindFirstMap(rec.ModifiedFields, func(fieldName appdef.FieldName, newValue interface{}) bool {
						return fieldName == appdef.SystemField_IsActive && newValue.(bool)
					})
					if activated {
						return true
					}
				}
			case appdef.ProjectorEventKind_Deactivate:
				if !rec.IsNew() {
					deactivated, _, _ := iterate.FindFirstMap(rec.ModifiedFields, func(fieldName appdef.FieldName, newValue interface{}) bool {
						return fieldName == appdef.SystemField_IsActive && !newValue.(bool)
					})
					if deactivated {
						return true
					}
				}
			}
		}
		return false
	})
	return triggered
}
