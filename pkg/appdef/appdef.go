/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"sort"
)

// # Implements:
//   - IAppDef
//   - IAppDefBuilder
type appDef struct {
	comment
	types        map[QName]interface{}
	typesOrdered []interface{}
}

func newAppDef() *appDef {
	app := appDef{
		types: make(map[QName]interface{}),
	}
	app.makeSysPackage()
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

func (app *appDef) AddData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	d := newData(app, name, kind, ancestor)
	d.AddConstraints(constraints...)
	app.appendType(d)
	return d
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

func (app *appDef) AddProjector(name QName) IProjectorBuilder {
	return newProjector(app, name)
}

func (app *appDef) AddQuery(name QName) IQueryBuilder {
	return newQuery(app, name)
}

func (app *appDef) AddSingleton(name QName) ICDocBuilder {
	doc := newCDoc(app, name)
	doc.SetSingleton()
	return doc
}

func (app *appDef) AddView(name QName) IViewBuilder {
	return newView(app, name)
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

func (app *appDef) Data(name QName) IData {
	if t := app.typeByKind(name, TypeKind_Data); t != nil {
		return t.(IData)
	}
	return nil
}

func (app *appDef) DataTypes(incSys bool, cb func(IData)) {
	app.Types(func(t IType) {
		if d, ok := t.(IData); ok {
			if incSys || !d.IsSystem() {
				cb(d)
			}
		}
	})
}

func (app *appDef) Functions(cb func(e IFunction)) {
	app.Types(func(t IType) {
		if f, ok := t.(IFunction); ok {
			cb(f)
		}
	})
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

func (app *appDef) Projector(name QName) IProjector {
	if t := app.typeByKind(name, TypeKind_Projector); t != nil {
		return t.(IProjector)
	}
	return nil
}

func (app *appDef) Projectors(cb func(IProjector)) {
	app.Types(func(t IType) {
		if p, ok := t.(IProjector); ok {
			cb(p)
		}
	})
}

func (app *appDef) Query(name QName) IQuery {
	if t := app.typeByKind(name, TypeKind_Query); t != nil {
		return t.(IQuery)
	}
	return nil
}

func (app *appDef) Record(name QName) IRecord {
	if t := app.TypeByName(name); t != nil {
		if r, ok := t.(IRecord); ok {
			return r
		}
	}
	return nil
}

func (app *appDef) Records(cb func(IRecord)) {
	app.Structures(func(s IStructure) {
		if r, ok := s.(IRecord); ok {
			cb(r)
		}
	})
}

func (app *appDef) Structures(cb func(s IStructure)) {
	app.Types(func(t IType) {
		if s, ok := t.(IStructure); ok {
			cb(s)
		}
	})
}

func (app *appDef) SysData(k DataKind) IData {
	if t := app.typeByKind(SysDataName(k), TypeKind_Data); t != nil {
		return t.(IData)
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

func (app *appDef) Types(cb func(IType)) {
	if app.typesOrdered == nil {
		app.typesOrdered = make([]interface{}, 0, len(app.types))
		for _, t := range app.types {
			app.typesOrdered = append(app.typesOrdered, t)
		}
		sort.Slice(app.typesOrdered, func(i, j int) bool {
			return app.typesOrdered[i].(IType).QName().String() < app.typesOrdered[j].(IType).QName().String()
		})
	}
	for _, t := range app.typesOrdered {
		cb(t.(IType))
	}
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

func (app *appDef) appendType(t interface{}) {
	typ := t.(IType)
	name := typ.QName()
	if name == NullQName {
		panic(fmt.Errorf("%s name cannot be empty: %w", typ.Kind().TrimString(), ErrNameMissed))
	}
	if app.TypeByName(name) != nil {
		panic(fmt.Errorf("type name «%s» already used: %w", name, ErrNameUniqueViolation))
	}

	app.types[name] = t
	app.typesOrdered = nil
}

// Makes system package.
//
// Should be called after appDef is created.
func (app *appDef) makeSysPackage() {
	app.makeSysDataTypes()
}

// Makes system data types.
func (app *appDef) makeSysDataTypes() {
	for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
		_ = newSysData(app, k)
	}
}

// Returns type by name and kind. If type is not found then returns nil.
func (app *appDef) typeByKind(name QName, kind TypeKind) interface{} {
	if t, ok := app.types[name]; ok {
		if t.(IType).Kind() == kind {
			return t
		}
	}
	return nil
}
