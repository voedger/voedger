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
type appDef struct {
	comment
	packages *packages
	sysWS    *workspace
	acl      []*aclRule // adding order should be saved
	types    *types[IType]
	wsDesc   map[QName]IWorkspace
}

func newAppDef() *appDef {
	app := appDef{
		packages: newPackages(),
		types:    newTypes[IType](),
		wsDesc:   make(map[QName]IWorkspace),
	}
	app.makeSysPackage()
	return &app
}

func (app appDef) ACL(cb func(IACLRule) bool) {
	for _, p := range app.acl {
		if !cb(p) {
			break
		}
	}
}

func (app appDef) FullQName(name QName) FullQName { return app.packages.fullQName(name) }

func (app appDef) LocalQName(name FullQName) QName { return app.packages.localQName(name) }

func (app *appDef) PackageLocalName(path string) string {
	return app.packages.localNameByPath(path)
}

func (app *appDef) PackageFullPath(local string) string {
	return app.packages.pathByLocalName(local)
}

func (app *appDef) PackageLocalNames() []string {
	return app.packages.localNames()
}

func (app *appDef) Packages(cb func(local, path string) bool) {
	app.packages.forEach(cb)
}

func (app *appDef) Type(name QName) IType {
	switch name {
	case NullQName:
		return NullType
	}

	if t, ok := anyTypes[name]; ok {
		return t
	}

	return app.types.find(name)
}

func (app *appDef) Types(visit func(IType) bool) {
	app.types.all(visit)
}

func (app *appDef) Workspace(name QName) IWorkspace {
	return TypeByNameAndKind[IWorkspace](app.Type, name, TypeKind_Workspace)
}

func (app *appDef) Workspaces(visit func(IWorkspace) bool) {
	TypesByKind[IWorkspace](app.Types, TypeKind_Workspace)(visit)
}

func (app *appDef) WorkspaceByDescriptor(name QName) IWorkspace {
	return app.wsDesc[name]
}

func (app *appDef) addPackage(localName, path string) {
	app.packages.add(localName, path)
}

func (app *appDef) addWorkspace(name QName) IWorkspaceBuilder {
	ws := newWorkspace(app, name)
	return newWorkspaceBuilder(ws)
}

func (app *appDef) alterWorkspace(name QName) IWorkspaceBuilder {
	w := app.Workspace(name)
	if w == nil {
		panic(ErrNotFound("workspace «%v»", name))
	}
	return newWorkspaceBuilder(w.(*workspace))
}

func (app *appDef) appendACL(p *aclRule) {
	app.acl = append(app.acl, p)
}

func (app *appDef) appendType(t IType) {
	name := t.QName()
	if name == NullQName {
		panic(ErrMissed("%s type name", t.Kind().TrimString()))
	}
	if app.Type(name).Kind() != TypeKind_null {
		panic(ErrAlreadyExists("type «%v»", name))
	}

	app.types.add(t)
}

func (app *appDef) build() (err error) {
	for t := range app.Types {
		err = errors.Join(err, validateType(t))
	}
	if err != nil {
		return err
	}
	for t := range app.Types {
		if b, ok := t.(interface{ build() error }); ok {
			err = errors.Join(err, b.build())
		}
	}
	return err
}

// Makes system package.
//
// Should be called after appDef is created.
func (app *appDef) makeSysPackage() {
	app.packages.add(SysPackage, SysPackagePath)
	app.makeSysWorkspace()
}

// Makes system workspace.
func (app *appDef) makeSysWorkspace() {
	app.sysWS = newWorkspace(app, SysWorkspaceQName)

	app.makeSysDataTypes()

	app.makeSysStructures()

	// TODO: move this code to sys.vsql (for projectors)
	viewProjectionOffsets := app.sysWS.addView(NewQName(SysPackage, "projectionOffsets"))
	viewProjectionOffsets.Key().PartKey().AddField("partition", DataKind_int32)
	viewProjectionOffsets.Key().ClustCols().AddField("projector", DataKind_QName)
	viewProjectionOffsets.Value().AddField("offset", DataKind_int64, true)

	// TODO: move this code to sys.vsql (for child workspaces)
	viewNextBaseWSID := app.sysWS.addView(NewQName(SysPackage, "NextBaseWSID"))
	viewNextBaseWSID.Key().PartKey().AddField("dummy1", DataKind_int32)
	viewNextBaseWSID.Key().ClustCols().AddField("dummy2", DataKind_int32)
	viewNextBaseWSID.Value().AddField("NextBaseWSID", DataKind_int64, true)
}

// Makes system data types.
func (app *appDef) makeSysDataTypes() {
	for k := DataKind_null + 1; k < DataKind_FakeLast; k++ {
		_ = newSysData(app, app.sysWS, k)
	}
}

func (app *appDef) makeSysStructures() {

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

func (ab *appDefBuilder) AddPackage(localName, path string) IAppDefBuilder {
	ab.app.addPackage(localName, path)
	return ab
}

func (ab *appDefBuilder) AddWorkspace(name QName) IWorkspaceBuilder { return ab.app.addWorkspace(name) }

func (ab *appDefBuilder) AlterWorkspace(name QName) IWorkspaceBuilder {
	return ab.app.alterWorkspace(name)
}

func (ab appDefBuilder) AppDef() IAppDef { return ab.app }

func (ab *appDefBuilder) Build() (IAppDef, error) {
	if err := ab.app.build(); err != nil {
		return nil, err
	}
	return ab.app, nil
}

func (ab *appDefBuilder) MustBuild() IAppDef {
	if err := ab.app.build(); err != nil {
		panic(err)
	}
	return ab.app
}
