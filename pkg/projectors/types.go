/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"sync"

	istructs "github.com/untillpro/voedger/pkg/istructs"
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

func isAcceptable(p istructs.Projector, event istructs.IPLogEvent) bool {
	if event.QName() == istructs.QNameForError {
		return p.HandleErrors
	}
	if len(p.EventsFilter) != 0 {
		for _, name := range p.EventsFilter {
			if name == event.QName() {
				return true
			}
		}
		return false
	}
	if len(p.EventsArgsFilter) != 0 {
		for _, name := range p.EventsArgsFilter {
			if name == event.ArgumentObject().QName() {
				return true
			}
		}
		return false
	}
	return true
}
