/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - IAppDef
//   - IAppDefBuilder
type appDef struct {
	changes int
	defs    map[QName]interface{}
}

func newAppDef() *appDef {
	app := appDef{
		defs: make(map[QName]interface{}),
	}
	return &app
}

func (app *appDef) AddCDoc(name QName) ICDocBuilder {
	return app.addDef(name, DefKind_CDoc)
}

func (app *appDef) AddSingleton(name QName) ICDocBuilder {
	d := app.addDef(name, DefKind_CDoc)
	d.SetSingleton()
	return d
}

func (app *appDef) AddCRecord(name QName) ICRecordBuilder {
	return app.addDef(name, DefKind_CRecord)
}

func (app *appDef) AddElement(name QName) IElementBuilder {
	return app.addDef(name, DefKind_Element)
}

func (app *appDef) AddGDoc(name QName) IGDocBuilder {
	return app.addDef(name, DefKind_GDoc)
}

func (app *appDef) AddGRecord(name QName) IGRecordBuilder {
	return app.addDef(name, DefKind_GRecord)
}

func (app *appDef) AddObject(name QName) IObjectBuilder {
	return app.addDef(name, DefKind_Object)
}

func (app *appDef) AddODoc(name QName) IODocBuilder {
	return app.addDef(name, DefKind_ODoc)
}

func (app *appDef) AddORecord(name QName) IORecordBuilder {
	return app.addDef(name, DefKind_ORecord)
}

func (app *appDef) AddWDoc(name QName) IWDocBuilder {
	return app.addDef(name, DefKind_WDoc)
}

func (app *appDef) AddView(name QName) IViewBuilder {
	v := newView(app, name)
	app.changed()
	return v
}

func (app *appDef) AddWRecord(name QName) IWRecordBuilder {
	return app.addDef(name, DefKind_WRecord)
}

func (app *appDef) Build() (result IAppDef, err error) {
	app.prepare()

	validator := newValidator()
	app.Defs(func(d IDef) {
		err = errors.Join(err, validator.validate(d))
	})
	if err != nil {
		return nil, err
	}

	app.changes = 0
	return app, nil
}

func (app *appDef) CDoc(name QName) (d ICDoc) {
	if d := app.defByKind(name, DefKind_CDoc); d != nil {
		return d.(ICDoc)
	}
	return nil
}

func (app *appDef) CRecord(name QName) ICRecord {
	if d := app.defByKind(name, DefKind_CRecord); d != nil {
		return d.(ICRecord)
	}
	return nil
}

func (app *appDef) Def(name QName) IDef {
	if d := app.DefByName(name); d != nil {
		return d
	}
	return NullDef
}

func (app *appDef) DefByName(name QName) IDef {
	if d, ok := app.defs[name]; ok {
		return d.(IDef)
	}
	return nil
}

func (app *appDef) DefCount() int {
	return len(app.defs)
}

func (app *appDef) Defs(cb func(IDef)) {
	for _, d := range app.defs {
		cb(d.(IDef))
	}
}

func (app *appDef) Element(name QName) IElement {
	if d := app.defByKind(name, DefKind_Element); d != nil {
		return d.(IElement)
	}
	return nil
}

func (app *appDef) GDoc(name QName) IGDoc {
	if d := app.defByKind(name, DefKind_GDoc); d != nil {
		return d.(IGDoc)
	}
	return nil
}

func (app *appDef) GRecord(name QName) IGRecord {
	if d := app.defByKind(name, DefKind_GRecord); d != nil {
		return d.(IGRecord)
	}
	return nil
}

func (app *appDef) HasChanges() bool {
	return app.changes > 0
}

func (app *appDef) Object(name QName) IObject {
	if d := app.defByKind(name, DefKind_Object); d != nil {
		return d.(IObject)
	}
	return nil
}

func (app *appDef) ODoc(name QName) IODoc {
	if d := app.defByKind(name, DefKind_ODoc); d != nil {
		return d.(IODoc)
	}
	return nil
}

func (app *appDef) ORecord(name QName) IORecord {
	if d := app.defByKind(name, DefKind_ORecord); d != nil {
		return d.(IORecord)
	}
	return nil
}

func (app *appDef) View(name QName) IView {
	if d := app.defByKind(name, DefKind_ViewRecord); d != nil {
		return d.(IView)
	}
	return nil
}

func (app *appDef) WDoc(name QName) IWDoc {
	if d := app.defByKind(name, DefKind_WDoc); d != nil {
		return d.(IWDoc)
	}
	return nil
}

func (app *appDef) WRecord(name QName) IWRecord {
	if d := app.defByKind(name, DefKind_WRecord); d != nil {
		return d.(IWRecord)
	}
	return nil
}

func (app *appDef) addDef(name QName, kind DefKind) *def {
	if name == NullQName {
		panic(fmt.Errorf("definition name cannot be empty: %w", ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("invalid definition name «%v»: %w", name, err))
	}
	if app.DefByName(name) != nil {
		panic(fmt.Errorf("definition name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	d := newDef(app, name, kind)
	app.appendDef(d)
	return d
}

func (app *appDef) appendDef(def interface{}) {
	app.defs[def.(IDef).QName()] = def
	app.changed()
}

func (app *appDef) changed() {
	app.changes++
}

func (app *appDef) defByKind(name QName, kind DefKind) interface{} {
	if d, ok := app.defs[name]; ok {
		if d.(IDef).Kind() == kind {
			return d
		}
	}
	return nil
}

func (app *appDef) prepare() {
}
