/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// # Implements:
//   - IProjector & IProjectorBuilder
type projector struct {
	typ
	sync bool
	ext  *extension
	//events    []*projectorEvent
	states  QNames
	intents QNames
}

// Returns new projector.
func newProjector(app *appDef, name QName) *projector {
	prj := &projector{
		typ: makeType(app, name, TypeKind_Projector),
		ext: newExtension(),
		//events: make([]*projectorEvent, 0),
		states:  QNames{},
		intents: QNames{},
	}
	app.appendType(prj)
	return prj
}

func (prj *projector) AddEvent(record QName, event ...ProjectorEventKind) IProjectorBuilder {
	// if e := prj.Event(record); e != nil {
	// 	e.addKind(event...)
	// } else {
	// 	prj.events = append(prj.events, newProjectorEvent(record, event...))
	// }
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

func (prj *projector) Events(func(IProjectorEvent)) {
	// for _, e := range prj.events {
	// 	fn(e)
	// }
}

func (prj *projector) Intents() QNames { return prj.intents }

func (prj *projector) SetEventComment(record QName, comment ...string) IProjectorBuilder {
	// e := prj.Event(record)
	// if e == nil {
	// 		panic(fmt.Errorf("%v: %v not found: %w", prj, record, ErrNameNotFound))
	// }
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
