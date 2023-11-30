/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 *
 * @author Michael Saigachenko
 */

package projectors

import (
	"context"
	"log"
	"slices"
	"sync"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
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

func isAcceptable(p istructs.Projector, event istructs.IPLogEvent, iprojector appdef.IProjector, triggeringQNames map[appdef.QName][]appdef.ProjectorEventKind) (bool, error) {
	if p.Name == appdef.NewQName("registry", "ProjectorLoginIdx") {
		log.Println()
	}
	if event.QName() == istructs.QNameForError {
		return iprojector.WantErrors(), nil
	}

	if triggeringKinds, ok := triggeringQNames[event.QName()]; ok {
		if slices.Contains(triggeringKinds, appdef.ProjectorEventKind_Execute) {
			return true, nil
		}
	}

	triggered := false
	event.CUDs(func(rec istructs.ICUDRow) error {
		if triggered {
			return nil
		}
		triggeringKinds, ok := triggeringQNames[rec.QName()]
		if !ok {
			return nil
		}
		for _, triggerkingKind := range triggeringKinds {
			switch triggerkingKind {
			case appdef.ProjectorEventKind_Insert:
				if rec.IsNew() {
					triggered = true
					return nil
				}
			case appdef.ProjectorEventKind_Update:
				if !rec.IsNew() {
					triggered = 
				}
			}
		}
	})


	iterate.FindFirst(iprojector.Events, func(pe appdef.IProjectorEvent) bool {
		// event's QName
		if event.QName() == pe.On().QName() {
			return true
		}

		// AFTER INSERT\UPDATE\ACTIVATE\DEACTIVATE
		err := event.CUDs(func(rec istructs.ICUDRow) error {
			pe.Kind()
		})
		if err != nil {
			// nolint
			return false, err
		}


	})

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
