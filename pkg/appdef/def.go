/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// NullDef is used for return then definition	is not founded
var NullDef = newDef(nil, NullQName, DefKind_null)

// # Implements:
//   - IDef and IDefBuilder
//   - IWithFields and IFieldsBuilder
//   - IWithContainers and IContainersBuilder
//   - IWithUniques and IUniquesBuilder
//   - IGDoc, IGDocBuilder and IGRecord, IGRecordBuilder
//   - ICDoc, ICDocBuilder and ICRecord, ICRecordBuilder
//   - IWDoc, IWDocBuilder and IWRecord, IWRecordBuilder
//   - IODoc, IODocBuilder and IORecord, IORecordBuilder
//   - IObject, IObjectBuilder and IElement, IElementBuilder
type def struct {
	app  *appDef
	name QName
	kind DefKind
	fields
	containers
	uniques
	singleton bool
	validate  func(*def) error
}

func makeDef(app *appDef, name QName, kind DefKind) def {
	def := def{
		app:  app,
		name: name,
		kind: kind,
	}
	def.fields = makeFields(&def)
	def.containers = makeContainers(&def)
	def.uniques = makeUniques(&def)
	return def
}

func newDef(app *appDef, name QName, kind DefKind) *def {
	def := makeDef(app, name, kind)
	return &def
}

func (d *def) App() IAppDef {
	return d.app
}

func (d *def) Kind() DefKind {
	return d.kind
}

func (d *def) QName() QName {
	return d.name
}

func (d *def) SetSingleton() {
	if d.Kind() != DefKind_CDoc {
		panic(fmt.Errorf("only CDocs can be singletons: %w", ErrInvalidDefKind))
	}
	d.singleton = true
	d.changed()
}

func (d *def) Singleton() bool {
	return d.singleton && (d.Kind() == DefKind_CDoc)
}

func (d *def) changed() {
	if d.app != nil {
		d.app.changed()
	}
}

func (d *def) Validate() error {
	if d.validate != nil {
		return d.validate(d)
	}
	return nil
}
