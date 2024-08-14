/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"sort"
)

// # Implements:
//   - IAppDef
type appDef struct {
	comment
	packages     *packages
	privileges   []*privilege // adding order should be saved
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

func (app *appDef) CDocs(cb func(ICDoc)) {
	app.Types(func(t IType) {
		if d, ok := t.(ICDoc); ok {
			cb(d)
		}
	})
}

func (app *appDef) Command(name QName) ICommand {
	if t := app.typeByKind(name, TypeKind_Command); t != nil {
		return t.(ICommand)
	}
	return nil
}

func (app *appDef) Commands(cb func(ICommand)) {
	app.Types(func(t IType) {
		if c, ok := t.(ICommand); ok {
			cb(c)
		}
	})
}

func (app *appDef) CRecord(name QName) ICRecord {
	if t := app.typeByKind(name, TypeKind_CRecord); t != nil {
		return t.(ICRecord)
	}
	return nil
}

func (app *appDef) CRecords(cb func(ICRecord)) {
	app.Types(func(t IType) {
		if r, ok := t.(ICRecord); ok {
			cb(r)
		}
	})
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

func (app *appDef) Extension(name QName) IExtension {
	if t := app.TypeByName(name); t != nil {
		if ex, ok := t.(IExtension); ok {
			return ex
		}
	}
	return nil
}

func (app *appDef) Extensions(cb func(IExtension)) {
	app.Types(func(t IType) {
		if ex, ok := t.(IExtension); ok {
			cb(ex)
		}
	})
}

func (app appDef) FullQName(name QName) FullQName { return app.packages.fullQName(name) }

func (app *appDef) GDoc(name QName) IGDoc {
	if t := app.typeByKind(name, TypeKind_GDoc); t != nil {
		return t.(IGDoc)
	}
	return nil
}

func (app *appDef) Function(name QName) IFunction {
	if t := app.TypeByName(name); t != nil {
		if f, ok := t.(IFunction); ok {
			return f
		}
	}
	return nil
}

func (app *appDef) Functions(cb func(IFunction)) {
	app.Types(func(t IType) {
		if f, ok := t.(IFunction); ok {
			cb(f)
		}
	})
}

func (app *appDef) GDocs(cb func(IGDoc)) {
	app.Types(func(t IType) {
		if d, ok := t.(IGDoc); ok {
			cb(d)
		}
	})
}

func (app *appDef) GRecord(name QName) IGRecord {
	if t := app.typeByKind(name, TypeKind_GRecord); t != nil {
		return t.(IGRecord)
	}
	return nil
}

func (app *appDef) GRecords(cb func(IGRecord)) {
	app.Types(func(t IType) {
		if r, ok := t.(IGRecord); ok {
			cb(r)
		}
	})
}

func (app *appDef) Job(name QName) IJob {
	if t := app.typeByKind(name, TypeKind_Job); t != nil {
		return t.(IJob)
	}
	return nil
}

func (app *appDef) Jobs(cb func(IJob)) {
	app.Types(func(t IType) {
		if j, ok := t.(IJob); ok {
			cb(j)
		}
	})
}

func (app *appDef) Limit(name QName) ILimit {
	if t := app.typeByKind(name, TypeKind_Limit); t != nil {
		return t.(ILimit)
	}
	return nil
}

func (app *appDef) Limits(cb func(ILimit)) {
	app.Types(func(t IType) {
		if l, ok := t.(ILimit); ok {
			cb(l)
		}
	})
}

func (app appDef) LocalQName(name FullQName) QName { return app.packages.localQName(name) }

func (app *appDef) Object(name QName) IObject {
	if t := app.typeByKind(name, TypeKind_Object); t != nil {
		return t.(IObject)
	}
	return nil
}

func (app *appDef) Objects(cb func(IObject)) {
	app.Types(func(t IType) {
		if o, ok := t.(IObject); ok {
			cb(o)
		}
	})
}

func (app *appDef) ODoc(name QName) IODoc {
	if t := app.typeByKind(name, TypeKind_ODoc); t != nil {
		return t.(IODoc)
	}
	return nil
}

func (app *appDef) ODocs(cb func(IODoc)) {
	app.Types(func(t IType) {
		if d, ok := t.(IODoc); ok {
			cb(d)
		}
	})
}

func (app *appDef) ORecord(name QName) IORecord {
	if t := app.typeByKind(name, TypeKind_ORecord); t != nil {
		return t.(IORecord)
	}
	return nil
}

func (app *appDef) ORecords(cb func(IORecord)) {
	app.Types(func(t IType) {
		if r, ok := t.(IORecord); ok {
			cb(r)
		}
	})
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

func (app appDef) Privileges(cb func(IPrivilege)) {
	for _, p := range app.privileges {
		cb(p)
	}
}

func (app appDef) PrivilegesOn(n []QName, k ...PrivilegeKind) []IPrivilege {
	pp := make([]IPrivilege, 0)
	for _, p := range app.privileges {
		if p.On().ContainsAny(n...) && p.kinds.ContainsAny(k...) {
			pp = append(pp, p)
		}
	}
	return pp
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

func (app *appDef) Queries(cb func(IQuery)) {
	app.Types(func(t IType) {
		if q, ok := t.(IQuery); ok {
			cb(q)
		}
	})
}

func (app *appDef) Query(name QName) IQuery {
	if t := app.typeByKind(name, TypeKind_Query); t != nil {
		return t.(IQuery)
	}
	return nil
}

func (app appDef) Rate(name QName) IRate {
	if t := app.typeByKind(name, TypeKind_Rate); t != nil {
		return t.(IRate)
	}
	return nil
}

func (app appDef) Rates(cb func(IRate)) {
	app.Types(func(t IType) {
		if r, ok := t.(IRate); ok {
			cb(r)
		}
	})
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

func (app *appDef) Role(name QName) IRole {
	if t := app.typeByKind(name, TypeKind_Role); t != nil {
		return t.(IRole)
	}
	return nil
}

func (app *appDef) Roles(cb func(IRole)) {
	app.Types(func(t IType) {
		if r, ok := t.(IRole); ok {
			cb(r)
		}
	})
}

func (app *appDef) Singleton(name QName) ISingleton {
	if t := app.TypeByName(name); t != nil {
		if s, ok := t.(ISingleton); ok {
			return s
		}
	}
	return nil
}

func (app *appDef) Singletons(cb func(ISingleton)) {
	app.Types(func(t IType) {
		if s, ok := t.(ISingleton); ok {
			cb(s)
		}
	})
}

func (app *appDef) Structure(name QName) IStructure {
	if t := app.TypeByName(name); t != nil {
		if s, ok := t.(IStructure); ok {
			return s
		}
	}
	return nil
}

func (app *appDef) Structures(cb func(IStructure)) {
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
	}

	if t, ok := anyTypes[name]; ok {
		return t
	}

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

func (app *appDef) Views(cb func(IView)) {
	app.Types(func(t IType) {
		if v, ok := t.(IView); ok {
			cb(v)
		}
	})
}

func (app *appDef) WDoc(name QName) IWDoc {
	if t := app.typeByKind(name, TypeKind_WDoc); t != nil {
		return t.(IWDoc)
	}
	return nil
}

func (app *appDef) WDocs(cb func(IWDoc)) {
	app.Types(func(t IType) {
		if d, ok := t.(IWDoc); ok {
			cb(d)
		}
	})
}

func (app *appDef) WRecord(name QName) IWRecord {
	if t := app.typeByKind(name, TypeKind_WRecord); t != nil {
		return t.(IWRecord)
	}
	return nil
}

func (app *appDef) WRecords(cb func(IWRecord)) {
	app.Types(func(t IType) {
		if r, ok := t.(IWRecord); ok {
			cb(r)
		}
	})
}

func (app *appDef) Workspace(name QName) IWorkspace {
	if t := app.typeByKind(name, TypeKind_Workspace); t != nil {
		return t.(IWorkspace)
	}
	return nil
}

func (app *appDef) Workspaces(cb func(IWorkspace)) {
	app.Types(func(t IType) {
		if ws, ok := t.(IWorkspace); ok {
			cb(ws)
		}
	})
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

func (app *appDef) addJob(name QName) IJobBuilder {
	j := newJob(app, name)
	return newJobBuilder(j)
}

func (app *appDef) addLimit(name QName, on []QName, rate QName, comment ...string) {
	_ = newLimit(app, name, on, rate, comment...)
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

func (app *appDef) addRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	_ = newRate(app, name, count, period, scopes, comment...)
}

func (app *appDef) addRole(name QName) IRoleBuilder {
	role := newRole(app, name)
	return newRoleBuilder(role)
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

func (app *appDef) appendPrivilege(p *privilege) {
	app.privileges = append(app.privileges, p)
}

func (app *appDef) appendType(t interface{}) {
	typ := t.(IType)
	name := typ.QName()
	if name == NullQName {
		panic(ErrMissed("%s type name", typ.Kind().TrimString()))
	}
	if app.TypeByName(name) != nil {
		panic(ErrAlreadyExists("type «%v»", name))
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

func (app *appDef) grant(kinds []PrivilegeKind, on []QName, fields []FieldName, toRole QName, comment ...string) {
	r := app.Role(toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	r.(*role).grant(kinds, on, fields, comment...)
}

func (app *appDef) grantAll(on []QName, toRole QName, comment ...string) {
	r := app.Role(toRole)
	if r == nil {
		panic(ErrRoleNotFound(toRole))
	}
	r.(*role).grantAll(on, comment...)
}

// Makes system package.
//
// Should be called after appDef is created.
func (app *appDef) makeSysPackage() {
	app.packages.add(SysPackage, SysPackagePath)
	app.makeSysDataTypes()
}

// Makes system data types.
func (app *appDef) makeSysDataTypes() {
	for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
		_ = newSysData(app, k)
	}
}

func (app *appDef) revoke(kinds []PrivilegeKind, on []QName, fromRole QName, comment ...string) {
	r := app.Role(fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	r.(*role).revoke(kinds, on, comment...)
}

func (app *appDef) revokeAll(on []QName, fromRole QName, comment ...string) {
	r := app.Role(fromRole)
	if r == nil {
		panic(ErrRoleNotFound(fromRole))
	}
	r.(*role).revokeAll(on, comment...)
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
	app                       *appDef
	hardcodedDefinitionsAdded bool
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

func (ab *appDefBuilder) AddJob(name QName) IJobBuilder { return ab.app.addJob(name) }

func (ab *appDefBuilder) AddLimit(name QName, on []QName, rate QName, comment ...string) {
	ab.app.addLimit(name, on, rate, comment...)
}

func (ab *appDefBuilder) AddObject(name QName) IObjectBuilder { return ab.app.addObject(name) }

func (ab *appDefBuilder) AddODoc(name QName) IODocBuilder { return ab.app.addODoc(name) }

func (ab *appDefBuilder) AddORecord(name QName) IORecordBuilder { return ab.app.addORecord(name) }

func (ab *appDefBuilder) AddPackage(localName, path string) IAppDefBuilder {
	ab.app.addPackage(localName, path)
	return ab
}

func (ab *appDefBuilder) AddProjector(name QName) IProjectorBuilder { return ab.app.addProjector(name) }

func (ab *appDefBuilder) AddQuery(name QName) IQueryBuilder { return ab.app.addQuery(name) }

func (ab *appDefBuilder) AddRate(name QName, count RateCount, period RatePeriod, scopes []RateScope, comment ...string) {
	ab.app.addRate(name, count, period, scopes, comment...)
}

func (ab *appDefBuilder) AddRole(name QName) IRoleBuilder { return ab.app.addRole(name) }

func (ab *appDefBuilder) AddView(name QName) IViewBuilder { return ab.app.addView(name) }

func (ab *appDefBuilder) AddWDoc(name QName) IWDocBuilder { return ab.app.addWDoc(name) }

func (ab *appDefBuilder) AddWRecord(name QName) IWRecordBuilder { return ab.app.addWRecord(name) }

func (ab *appDefBuilder) AddWorkspace(name QName) IWorkspaceBuilder { return ab.app.addWorkspace(name) }

func (ab appDefBuilder) AppDef() IAppDef { return ab.app }

func (ab *appDefBuilder) Build() (IAppDef, error) {
	if !ab.hardcodedDefinitionsAdded {
		ab.addHardcodedDefinitions()
		ab.hardcodedDefinitionsAdded = true
	}
	if err := ab.app.build(); err != nil {
		return nil, err
	}
	return ab.app, nil
}

func (ab *appDefBuilder) addHardcodedDefinitions() {
	viewProjectionOffsets := ab.AddView(NewQName(SysPackage, "projectionOffsets"))
	viewProjectionOffsets.Key().PartKey().AddField("partition", DataKind_int32)
	viewProjectionOffsets.Key().ClustCols().AddField("projector", DataKind_QName)
	viewProjectionOffsets.Value().AddField("offset", DataKind_int64, true)

	viewNextBaseWSID := ab.AddView(NewQName(SysPackage, "NextBaseWSID"))
	viewNextBaseWSID.Key().PartKey().AddField("dummy1", DataKind_int32)
	viewNextBaseWSID.Key().ClustCols().AddField("dummy2", DataKind_int32)
	viewNextBaseWSID.Value().AddField("NextBaseWSID", DataKind_int64, true)
}

func (ab *appDefBuilder) Grant(kinds []PrivilegeKind, on []QName, fields []FieldName, toRole QName, comment ...string) IPrivilegesBuilder {
	ab.app.grant(kinds, on, fields, toRole, comment...)
	return ab
}

func (ab *appDefBuilder) GrantAll(on []QName, toRole QName, comment ...string) IPrivilegesBuilder {
	ab.app.grantAll(on, toRole, comment...)
	return ab
}

func (ab *appDefBuilder) MustBuild() IAppDef {
	if err := ab.app.build(); err != nil {
		panic(err)
	}
	return ab.app
}

func (ab *appDefBuilder) Revoke(kinds []PrivilegeKind, on []QName, fromRole QName, comment ...string) IPrivilegesBuilder {
	ab.app.revoke(kinds, on, fromRole, comment...)
	return ab
}

func (ab *appDefBuilder) RevokeAll(on []QName, fromRole QName, comment ...string) IPrivilegesBuilder {
	ab.app.revokeAll(on, fromRole, comment...)
	return ab
}
