/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package actualizers

import (
	"context"
	"fmt"
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

// FIXME: should accept appdef.IProjector instead triggers.
func isAcceptable(event istructs.IPLogEvent, wantErrors bool, triggers map[appdef.QName]appdef.OperationsSet, _ appdef.IAppDef, projQName appdef.QName) (ok bool) {
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

	if len(triggers) == 0 {
		return true //nnv: Suspicious
	}

	if ops, ok := triggers[event.QName()]; ok {
		// Event QName is Command `ON EXECUTE`
		if ops.Contains(appdef.OperationKind_Execute) {
			return true
		}
		// Event QName is ODoc `ON EXECUTE WITH`
		if ops.Contains(appdef.OperationKind_ExecuteWithParam) {
			return true
		}
	}

	if ops, ok := triggers[event.ArgumentObject().QName()]; ok {
		// Event Arg is ODoc or Object `ON EXECUTE WITH`
		if ops.Contains(appdef.OperationKind_ExecuteWithParam) {
			return true
		}
	}

	for rec := range event.CUDs {
		if ops, ok := triggers[rec.QName()]; ok {
			for op := range ops.Values() {
				switch op {
				case appdef.OperationKind_Insert:
					if rec.IsNew() {
						return true
					}
				case appdef.OperationKind_Update:
					if !rec.IsNew() {
						return true
					}
				case appdef.OperationKind_Activate:
					if !rec.IsNew() {
						for fieldName, newValue := range rec.ModifiedFields {
							if fieldName == appdef.SystemField_IsActive && newValue.(bool) {
								return true
							}
						}
					}
				case appdef.OperationKind_Deactivate:
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
	}
	return false
}
