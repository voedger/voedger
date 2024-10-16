/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
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

func isAcceptable(event istructs.IPLogEvent, wantErrors bool, triggeringQNames map[appdef.QName][]appdef.ProjectorEventKind, appDef appdef.IAppDef, projQName appdef.QName) (ok bool) {
	defer func() {
		if ok && logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("projector %s is acceptable to event %s", projQName, event.QName()))
		}
	}()
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

	for rec := range event.CUDs {
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
					for fieldName, newValue := range rec.ModifiedFields {
						if fieldName == appdef.SystemField_IsActive && newValue.(bool) {
							return true
						}
					}
				}
			case appdef.ProjectorEventKind_Deactivate:
				if !rec.IsNew() {
					for fieldName, newValue := range rec.ModifiedFields {
						if fieldName == appdef.SystemField_IsActive && !newValue.(bool) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
