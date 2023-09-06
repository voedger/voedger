/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
)

// # Implements:
//   - IAppDef
//   - IAppDefBuilder
type appDef struct {
	comment
	defs map[QName]interface{}
}

func newAppDef() *appDef {
	app := appDef{
		defs: make(map[QName]interface{}),
	}
	return &app
}

func (app *appDef) AddCDoc(name QName) ICDocBuilder {
	return newCDoc(app, name)
}

func (app *appDef) AddCommand(name QName) ICommandBuilder {
	return newCommand(app, name)
}

func (app *appDef) AddCRecord(name QName) ICRecordBuilder {
	return newCRecord(app, name)
}

func (app *appDef) AddElement(name QName) IElementBuilder {
	return newElement(app, name)
}

func (app *appDef) AddGDoc(name QName) IGDocBuilder {
	return newGDoc(app, name)
}

func (app *appDef) AddGRecord(name QName) IGRecordBuilder {
	return newGRecord(app, name)
}

func (app *appDef) AddObject(name QName) IObjectBuilder {
	return newObject(app, name)
}

func (app *appDef) AddODoc(name QName) IODocBuilder {
	return newODoc(app, name)
}

func (app *appDef) AddORecord(name QName) IORecordBuilder {
	return newORecord(app, name)
}

func (app *appDef) AddSingleton(name QName) ICDocBuilder {
	doc := newCDoc(app, name)
	doc.SetSingleton()
	return doc
}

func (app *appDef) AddQuery(name QName) IQueryBuilder {
	return newQuery(app, name)
}

func (app *appDef) AddView(name QName) IViewBuilder {
	return newViewBuilder(app, name)
}

func (app *appDef) AddWDoc(name QName) IWDocBuilder {
	return newWDoc(app, name)
}

func (app *appDef) AddWRecord(name QName) IWRecordBuilder {
	return newWRecord(app, name)
}

func (app *appDef) AddWorkspace(name QName) IWorkspaceBuilder {
	return newWorkspace(app, name)
}

func (app *appDef) Build() (result IAppDef, err error) {
	app.Defs(func(d IDef) {
		err = errors.Join(err, validateDef(d))
	})
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *appDef) CDoc(name QName) (d ICDoc) {
	if d := app.defByKind(name, DefKind_CDoc); d != nil {
		return d.(ICDoc)
	}
	return nil
}

func (app *appDef) Command(name QName) ICommand {
	if d := app.defByKind(name, DefKind_Command); d != nil {
		return d.(ICommand)
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

func (app *appDef) Query(name QName) IQuery {
	if d := app.defByKind(name, DefKind_Query); d != nil {
		return d.(IQuery)
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

func (app *appDef) Workspace(name QName) IWorkspace {
	if d := app.defByKind(name, DefKind_Workspace); d != nil {
		return d.(IWorkspace)
	}
	return nil
}

func (app *appDef) appendDef(def interface{}) {
	app.defs[def.(IDef).QName()] = def
}

func (app *appDef) defByKind(name QName, kind DefKind) interface{} {
	if d, ok := app.defs[name]; ok {
		if d.(IDef).Kind() == kind {
			return d
		}
	}
	return nil
}
