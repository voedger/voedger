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

// Returns is projector triggered by event
func isAcceptable(prj appdef.IProjector, event istructs.IPLogEvent) (ok bool) {
	defer func() {
		if ok && logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("%v is triggered by %v", prj, event))
		}
	}()

	switch event.QName() {
	case istructs.QNameForError:
		return prj.WantErrors()
	case istructs.QNameForCorruptedData:
		return false
	}

	if cmd := appdef.Command(prj.App().Type, event.QName()); cmd != nil {
		if prj.Triggers(appdef.OperationKind_Execute, cmd) {
			return true // ON EXECUTE <Command>
		}
	}

	if doc := appdef.ODoc(prj.App().Type, event.QName()); doc != nil {
		if prj.Triggers(appdef.OperationKind_ExecuteWithParam, doc) {
			return true // ON EXECUTE WITH <ODoc>
		}
		if prj.Triggers(appdef.OperationKind_Execute, doc) {
			return true // ON EXECUTE <ODoc>
		}
	}

	if arg := prj.App().Type(event.ArgumentObject().QName()); arg.Kind() != appdef.TypeKind_null {
		if prj.Triggers(appdef.OperationKind_ExecuteWithParam, arg) {
			return true // ON EXECUTE WITH <ODoc>; ON EXECUTE WITH <Object>
		}
	}

	for rec := range event.CUDs {
		t := prj.App().Type(rec.QName())
		if t.Kind() != appdef.TypeKind_null {
			if rec.IsNew() {
				if prj.Triggers(appdef.OperationKind_Insert, t) {
					return true // ON INSERT <Record>
				}
			} else {
				if prj.Triggers(appdef.OperationKind_Update, t) {
					return true // ON UPDATE <Record>
				}
				if rec.IsDeactivated() {
					if prj.Triggers(appdef.OperationKind_Deactivate, t) {
						return true // ON DEACTIVATE <Record>
					}
				} else if rec.IsActivated() {
					if prj.Triggers(appdef.OperationKind_Activate, t) {
						return true // ON ACTIVATE <Record>
					}
				}
			}
		}
	}
	return false
}
