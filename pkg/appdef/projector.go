/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strings"
)

// # Implements:
//   - IProjector & IProjectorBuilder
type projector struct {
	typ
	sync    bool
	ext     *extension
	events  map[QName]*projectorEvent
	states  QNames
	intents QNames
}

// Returns new projector.
func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		typ:     makeType(app, name, TypeKind_Projector),
		ext:     newExtension(),
		events:  make(map[QName]*projectorEvent),
		states:  QNames{},
		intents: QNames{},
	}
	app.appendType(prj)
	return prj
}

func (prj *projector) AddEvent(record QName, event ...ProjectorEventKind) IProjectorBuilder {
	if record == NullQName {
		panic(fmt.Errorf("%v: can not add event because record name is empty: %w", prj, ErrNameMissed))
	}

	rec := func() (rec IType) {
		switch record {
		case NullQName:
			panic(fmt.Errorf("%v: can not add event because record name is empty: %w", prj, ErrNameMissed))
		case QNameANY:
			rec = AnyType
		default:
			rec = prj.app.Record(record)
			if rec == nil {
				panic(fmt.Errorf("%v: record type «%v» not found: %w", prj, record, ErrNameNotFound))
			}
		}
		return rec
	}()

	if e, ok := prj.events[record]; ok {
		e.addKind(event...)
	} else {
		prj.events[record] = newProjectorEvent(rec, event...)
	}
	return prj
}

func (prj *projector) AddState(states ...QName) IProjectorBuilder {
	prj.states.Append(states...)
	return prj
}

func (prj *projector) AddIntent(intents ...QName) IProjectorBuilder {
	prj.intents.Append(intents...)
	return prj
}

func (prj *projector) Extension() IExtension { return prj.ext }

func (prj *projector) Events(cb func(IProjectorEvent)) {
	ord := QNames{}
	for n := range prj.events {
		ord.Append(n)
	}
	for _, n := range ord {
		cb(prj.events[n])
	}
}

func (prj *projector) Intents() QNames { return prj.intents }

func (prj *projector) SetEventComment(record QName, comment ...string) IProjectorBuilder {
	e, ok := prj.events[record]
	if !ok {
		panic(fmt.Errorf("%v: %v not found: %w", prj, record, ErrNameNotFound))
	}
	e.SetComment(comment...)
	return prj
}

func (prj *projector) SetExtension(name string, engine ExtensionEngineKind, comment ...string) IProjectorBuilder {
	if name == "" {
		name = prj.QName().Entity()
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", prj, name, err))
	}
	prj.ext.name = name
	prj.ext.engine = engine
	prj.ext.SetComment(comment...)

	return prj
}

func (prj *projector) SetSync(sync bool) IProjectorBuilder {
	prj.sync = sync
	return prj
}

func (prj *projector) Sync() bool { return prj.sync }

func (prj *projector) States() QNames { return prj.states }

type projectorEvent struct {
	comment
	record IType
	kinds  uint64 // bitmap[ProjectorEventKind]
}

func newProjectorEvent(record IType, kind ...ProjectorEventKind) *projectorEvent {
	p := &projectorEvent{record: record}
	p.addKind(kind...)
	return p
}

func (e *projectorEvent) Kind() (kinds []ProjectorEventKind) {
	for k := ProjectorEventKind(1); k < ProjectorEventKind_Count; k++ {
		if e.kinds&(1<<k) != 0 {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

func (e *projectorEvent) On() IType {
	return e.record
}

func (e projectorEvent) String() string {
	s := []string{}
	for _, k := range e.Kind() {
		s = append(s, k.TrimString())
	}
	return fmt.Sprintf("%v [%s]", e.On(), strings.Join(s, " "))
}

func (e *projectorEvent) addKind(kind ...ProjectorEventKind) {
	for _, k := range kind {
		e.kinds |= 1 << k
	}
}
