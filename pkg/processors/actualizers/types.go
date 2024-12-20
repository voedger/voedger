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
func ProjectorEvent(prj appdef.IProjector, event istructs.IPLogEvent) (triggered bool) {
	defer func() {
		if triggered && logger.IsVerbose() {
			logger.Verbose(fmt.Sprintf("%v is triggered by %v", prj, event))
		}
	}()

	switch event.QName() {
	case istructs.QNameForError:
		return prj.WantErrors()
	case istructs.QNameForCorruptedData:
		return false
	}

	t := prj.App().Type(event.QName())
	if prj.Triggers(appdef.OperationKind_Execute, t) {
		return true // ON EXECUTE <Command> || ON EXECUTE <ODoc>
	}

	if arg := event.ArgumentObject().QName(); arg != appdef.NullQName {
		t := prj.App().Type(arg)
		if prj.Triggers(appdef.OperationKind_ExecuteWithParam, t) {
			return true // ON EXECUTE WITH <ODoc>; ON EXECUTE WITH <Object>
		}
	}

	for rec := range event.CUDs {
		t := prj.App().Type(rec.QName())
		if rec.IsNew() {
			if prj.Triggers(appdef.OperationKind_Insert, t) {
				return true // AFTER INSERT <Record>
			}
		} else {
			if prj.Triggers(appdef.OperationKind_Update, t) {
				return true // AFTER UPDATE <Record>
			}
			if rec.IsDeactivated() {
				if prj.Triggers(appdef.OperationKind_Deactivate, t) {
					return true // AFTER DEACTIVATE <Record>
				}
			} else if rec.IsActivated() {
				if prj.Triggers(appdef.OperationKind_Activate, t) {
					return true // AFTER ACTIVATE <Record>
				}
			}
		}
	}
	return false
}
