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
	types map[QName]interface{}
}

func newAppDef() *appDef {
	app := appDef{
		types: make(map[QName]interface{}),
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
	app.Types(func(t IType) {
		err = errors.Join(err, validateType(t))
	})
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *appDef) CDoc(name QName) (d ICDoc) {
	if t := app.typeByKind(name, TypeKind_CDoc); t != nil {
		return t.(ICDoc)
	}
	return nil
}

func (app *appDef) Command(name QName) ICommand {
	if t := app.typeByKind(name, TypeKind_Command); t != nil {
		return t.(ICommand)
	}
	return nil
}

func (app *appDef) CRecord(name QName) ICRecord {
	if t := app.typeByKind(name, TypeKind_CRecord); t != nil {
		return t.(ICRecord)
	}
	return nil
}

func (app *appDef) Type(name QName) IType {
	if t := app.TypeByName(name); t != nil {
		return t
	}
	return NullType
}

func (app *appDef) TypeByName(name QName) IType {
	if t, ok := app.types[name]; ok {
		return t.(IType)
	}
	return nil
}

func (app *appDef) TypeCount() int {
	return len(app.types)
}

func (app *appDef) Types(cb func(IType)) {
	for _, t := range app.types {
		cb(t.(IType))
	}
}

func (app *appDef) Element(name QName) IElement {
	if t := app.typeByKind(name, TypeKind_Element); t != nil {
		return t.(IElement)
	}
	return nil
}

func (app *appDef) GDoc(name QName) IGDoc {
	if t := app.typeByKind(name, TypeKind_GDoc); t != nil {
		return t.(IGDoc)
	}
	return nil
}

func (app *appDef) GRecord(name QName) IGRecord {
	if t := app.typeByKind(name, TypeKind_GRecord); t != nil {
		return t.(IGRecord)
	}
	return nil
}

func (app *appDef) Object(name QName) IObject {
	if t := app.typeByKind(name, TypeKind_Object); t != nil {
		return t.(IObject)
	}
	return nil
}

func (app *appDef) ODoc(name QName) IODoc {
	if t := app.typeByKind(name, TypeKind_ODoc); t != nil {
		return t.(IODoc)
	}
	return nil
}

func (app *appDef) ORecord(name QName) IORecord {
	if t := app.typeByKind(name, TypeKind_ORecord); t != nil {
		return t.(IORecord)
	}
	return nil
}

func (app *appDef) Query(name QName) IQuery {
	if t := app.typeByKind(name, TypeKind_Query); t != nil {
		return t.(IQuery)
	}
	return nil
}

func (app *appDef) View(name QName) IView {
	if t := app.typeByKind(name, TypeKind_ViewRecord); t != nil {
		return t.(IView)
	}
	return nil
}

func (app *appDef) WDoc(name QName) IWDoc {
	if t := app.typeByKind(name, TypeKind_WDoc); t != nil {
		return t.(IWDoc)
	}
	return nil
}

func (app *appDef) WRecord(name QName) IWRecord {
	if t := app.typeByKind(name, TypeKind_WRecord); t != nil {
		return t.(IWRecord)
	}
	return nil
}

func (app *appDef) Workspace(name QName) IWorkspace {
	if t := app.typeByKind(name, TypeKind_Workspace); t != nil {
		return t.(IWorkspace)
	}
	return nil
}

func (app *appDef) appendType(def interface{}) {
	app.types[def.(IType).QName()] = def
}

func (app *appDef) typeByKind(name QName, kind TypeKind) interface{} {
	if t, ok := app.types[name]; ok {
		if t.(IType).Kind() == kind {
			return t
		}
	}
	return nil
}
