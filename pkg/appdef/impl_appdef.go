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
type appDef struct {
	comment
	packages     *packages
	types        map[QName]interface{}
	typesOrdered []interface{}
	wsDesc       map[QName]IWorkspace
}

func newAppDef() *appDef {
	app := appDef{
		packages: newPackages(),
		types:    make(map[QName]interface{}),
		wsDesc:   make(map[QName]IWorkspace),
	}
	app.makeSysPackage()
	return &app
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

func (app *appDef) Extensions(cb func(e IExtension)) {
	app.Types(func(t IType) {
		if ex, ok := t.(IExtension); ok {
			cb(ex)
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

func (app *appDef) PackageLocalName(path string) string {
	return app.packages.localNameByPath(path)
}

func (app *appDef) PackageFullPath(local string) string {
	return app.packages.pathByLocalName(local)
}

func (app *appDef) PackageLocalNames() []string {
	return app.packages.localNames()
}

func (app *appDef) Packages(cb func(local, path string)) {
	app.packages.forEach(cb)
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
	switch name {
	case NullQName:
		return NullType
	case QNameANY:
		return AnyType
	default:
		if t, ok := app.types[name]; ok {
			return t.(IType)
		}
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

func (app *appDef) WorkspaceByDescriptor(name QName) IWorkspace {
	return app.wsDesc[name]
}

func (app *appDef) addCDoc(name QName) ICDocBuilder {
	cDoc := newCDoc(app, name)
	return newCDocBuilder(cDoc)
}

func (app *appDef) addCommand(name QName) ICommandBuilder {
	cmd := newCommand(app, name)
	return newCommandBuilder(cmd)
}

func (app *appDef) addCRecord(name QName) ICRecordBuilder {
	cRec := newCRecord(app, name)
	return newCRecordBuilder(cRec)
}

func (app *appDef) addData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	d := newData(app, name, kind, ancestor)
	d.addConstraints(constraints...)
	app.appendType(d)
	return newDataBuilder(d)
}

func (app *appDef) addGDoc(name QName) IGDocBuilder {
	gDoc := newGDoc(app, name)
	return newGDocBuilder(gDoc)
}

func (app *appDef) addGRecord(name QName) IGRecordBuilder {
	gRec := newGRecord(app, name)
	return newGRecordBuilder(gRec)
}

func (app *appDef) addObject(name QName) IObjectBuilder {
	obj := newObject(app, name)
	return newObjectBuilder(obj)
}

func (app *appDef) addODoc(name QName) IODocBuilder {
	oDoc := newODoc(app, name)
	return newODocBuilder(oDoc)
}

func (app *appDef) addORecord(name QName) IORecordBuilder {
	oRec := newORecord(app, name)
	return newORecordBuilder(oRec)
}

func (app *appDef) addPackage(localName, path string) {
	app.packages.add(localName, path)
}

func (app *appDef) addProjector(name QName) IProjectorBuilder {
	projector := newProjector(app, name)
	return newProjectorBuilder(projector)
}

func (app *appDef) addQuery(name QName) IQueryBuilder {
	q := newQuery(app, name)
	return newQueryBuilder(q)
}

func (app *appDef) addView(name QName) IViewBuilder {
	view := newView(app, name)
	return newViewBuilder(view)
}

func (app *appDef) addWDoc(name QName) IWDocBuilder {
	wDoc := newWDoc(app, name)
	return newWDocBuilder(wDoc)
}

func (app *appDef) addWRecord(name QName) IWRecordBuilder {
	wRec := newWRecord(app, name)
	return newWRecordBuilder(wRec)
}

func (app *appDef) addWorkspace(name QName) IWorkspaceBuilder {
	ws := newWorkspace(app, name)
	return newWorkspaceBuilder(ws)
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

func (app *appDef) build() (err error) {
	app.Types(func(t IType) {
		err = errors.Join(err, validateType(t))
	})
	return err
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

// # Implements:
//   - IAppDefBuilder
type appDefBuilder struct {
	commentBuilder
	app *appDef
}

func newAppDefBuilder(app *appDef) *appDefBuilder {
	return &appDefBuilder{
		commentBuilder: makeCommentBuilder(&app.comment),
		app:            app,
	}
}

func (ab *appDefBuilder) AddCDoc(name QName) ICDocBuilder { return ab.app.addCDoc(name) }

func (ab *appDefBuilder) AddCommand(name QName) ICommandBuilder { return ab.app.addCommand(name) }

func (ab *appDefBuilder) AddCRecord(name QName) ICRecordBuilder { return ab.app.addCRecord(name) }

func (ab *appDefBuilder) AddData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder {
	return ab.app.addData(name, kind, ancestor, constraints...)
}

func (ab *appDefBuilder) AddGDoc(name QName) IGDocBuilder { return ab.app.addGDoc(name) }

func (ab *appDefBuilder) AddGRecord(name QName) IGRecordBuilder { return ab.app.addGRecord(name) }

func (ab *appDefBuilder) AddObject(name QName) IObjectBuilder { return ab.app.addObject(name) }

func (ab *appDefBuilder) AddODoc(name QName) IODocBuilder { return ab.app.addODoc(name) }

func (ab *appDefBuilder) AddORecord(name QName) IORecordBuilder { return ab.app.addORecord(name) }

func (ab *appDefBuilder) AddPackage(localName, path string) IAppDefBuilder {
	ab.app.addPackage(localName, path)
	return ab
}

func (ab *appDefBuilder) AddProjector(name QName) IProjectorBuilder { return ab.app.addProjector(name) }

func (ab *appDefBuilder) AddQuery(name QName) IQueryBuilder { return ab.app.addQuery(name) }

func (ab *appDefBuilder) AddView(name QName) IViewBuilder { return ab.app.addView(name) }

func (ab *appDefBuilder) AddWDoc(name QName) IWDocBuilder { return ab.app.addWDoc(name) }

func (ab *appDefBuilder) AddWRecord(name QName) IWRecordBuilder { return ab.app.addWRecord(name) }

func (ab *appDefBuilder) AddWorkspace(name QName) IWorkspaceBuilder { return ab.app.addWorkspace(name) }

func (ab appDefBuilder) AppDef() IAppDef { return ab.app }

func (ab *appDefBuilder) Build() (IAppDef, error) {
	if err := ab.app.build(); err != nil {
		return nil, err
	}
	return ab.app, nil
}
