/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apps

import (
	"errors"
	"iter"
	"maps"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/containers"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
	"github.com/voedger/voedger/pkg/appdef/internal/fields"
	"github.com/voedger/voedger/pkg/appdef/internal/packages"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/appdef/internal/workspaces"
)

// # Supports:
//   - appdef.IAppDef
type AppDef struct {
	comments.WithComments
	packages   *packages.Packages
	sysWS      *workspaces.Workspace
	acl        []appdef.IACLRule
	types      *types.Types[appdef.IType]
	workspaces *workspaces.Workspaces
	wsDesc     map[appdef.QName]appdef.IWorkspace
}

func NewAppDef() *AppDef {
	app := AppDef{
		WithComments: comments.MakeWithComments(),
		packages:     packages.NewPackages(),
		acl:          make([]appdef.IACLRule, 0),
		types:        types.NewTypes[appdef.IType](),
		workspaces:   workspaces.NewWorkspaces(),
		wsDesc:       make(map[appdef.QName]appdef.IWorkspace),
	}
	app.makeSysPackage()
	return &app
}

func (app AppDef) ACL() iter.Seq[appdef.IACLRule] { return slices.Values(app.acl) }

func (app *AppDef) AppendACL(acl appdef.IACLRule) {
	app.acl = append(app.acl, acl)
}

func (app *AppDef) AppendType(t appdef.IType) {
	name := t.QName()
	if name == appdef.NullQName {
		panic(appdef.ErrMissed("%s type name", t.Kind().TrimString()))
	}
	if app.Type(name).Kind() != appdef.TypeKind_null {
		panic(appdef.ErrAlreadyExists("type «%v»", name))
	}

	app.types.Add(t)
}

func (app AppDef) FullQName(name appdef.QName) appdef.FullQName { return app.packages.FullQName(name) }

func (app AppDef) LocalQName(name appdef.FullQName) appdef.QName {
	return app.packages.LocalQName(name)
}

func (app AppDef) PackageLocalName(path string) string {
	return app.packages.LocalNameByPath(path)
}

func (app AppDef) PackageFullPath(local string) string {
	return app.packages.PathByLocalName(local)
}

func (app AppDef) PackageLocalNames() iter.Seq[string] {
	return app.packages.PackageLocalNames()
}

func (app AppDef) Packages() iter.Seq2[string, string] { return app.packages.Packages() }

func (app *AppDef) SetWorkspaceDescriptor(ws, desc appdef.QName) {
	maps.DeleteFunc(app.wsDesc, func(_ appdef.QName, w appdef.IWorkspace) bool {
		return w.QName() == ws // remove old descriptor
	})
	app.wsDesc[desc] = app.Workspace(ws)
}

func (app AppDef) Type(name appdef.QName) appdef.IType {
	switch name {
	case appdef.NullQName:
		return appdef.NullType
	case appdef.QNameANY:
		return appdef.AnyType
	}
	return app.types.Find(name)
}

func (app AppDef) Types() iter.Seq[appdef.IType] { return app.types.Values() }

func (app *AppDef) Workspace(name appdef.QName) appdef.IWorkspace {
	w := app.workspaces.Find(name)
	if w != appdef.NullType {
		return w.(appdef.IWorkspace)
	}
	return nil
}

func (app AppDef) Workspaces() iter.Seq[appdef.IWorkspace] { return app.workspaces.Values() }

func (app AppDef) WorkspaceByDescriptor(name appdef.QName) appdef.IWorkspace {
	return app.wsDesc[name]
}

func (app *AppDef) addPackage(localName, path string) {
	app.packages.Add(localName, path)
}

func (app *AppDef) addWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	ws := workspaces.NewWorkspace(app, name)
	app.workspaces.Add(ws)
	return workspaces.NewWorkspaceBuilder(ws)
}

func (app *AppDef) alterWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	ws := app.Workspace(name)
	if ws == nil {
		panic(appdef.ErrNotFound("workspace «%v»", name))
	}
	return workspaces.NewWorkspaceBuilder(ws.(*workspaces.Workspace))
}

func (app *AppDef) build() (err error) {
	for t := range app.Types() {
		err = errors.Join(err, app.validateType(t))
	}
	return err
}

// Makes system package.
//
// Should be called after appDef is created.
func (app *AppDef) makeSysPackage() {
	app.packages.Add(appdef.SysPackage, appdef.SysPackagePath)
	app.makeSysWorkspace()
}

// Makes system workspace.
func (app *AppDef) makeSysWorkspace() {
	app.sysWS = workspaces.NewWorkspace(app, appdef.SysWorkspaceQName)
	app.workspaces.Add(app.sysWS)

	app.makeSysDataTypes()

	app.makeSysStructures()
}

// Makes system data types.
func (app *AppDef) makeSysDataTypes() {
	for k := appdef.DataKind_null + 1; k < appdef.DataKind_FakeLast; k++ {
		d := datas.NewSysData(app.sysWS, k)
		app.sysWS.AppendType(d) // propagate type to ws and app
	}
}

func (app *AppDef) makeSysStructures() {
	wsb := workspaces.NewWorkspaceBuilder(app.sysWS)
	// TODO: move this code to sys.vsql (for projectors)
	viewProjectionOffsets := wsb.AddView(appdef.NewQName(appdef.SysPackage, "projectionOffsets"))
	viewProjectionOffsets.Key().PartKey().AddField("partition", appdef.DataKind_int32)
	viewProjectionOffsets.Key().ClustCols().AddField("projector", appdef.DataKind_QName)
	viewProjectionOffsets.Value().AddField("offset", appdef.DataKind_int64, true)

	// TODO: move this code to sys.vsql (for child workspaces)
	viewNextBaseWSID := wsb.AddView(appdef.NewQName(appdef.SysPackage, "NextBaseWSID"))
	viewNextBaseWSID.Key().PartKey().AddField("dummy1", appdef.DataKind_int32)
	viewNextBaseWSID.Key().ClustCols().AddField("dummy2", appdef.DataKind_int32)
	viewNextBaseWSID.Value().AddField("NextBaseWSID", appdef.DataKind_int64, true)
}

func (app *AppDef) setTypeComment(n appdef.QName, comment ...string) {
	t := app.Type(n)
	if t == appdef.NullType {
		panic(appdef.ErrNotFound("type %v", n))
	}

	if t, ok := t.(*types.Typ); ok {
		comments.SetComment(&t.WithComments, comment...)
	}
}

func (app *AppDef) validateType(t appdef.IType) (err error) {
	if v, ok := t.(interface{ Validate() error }); ok {
		err = v.Validate()
	}

	if _, ok := t.(appdef.IFields); ok {
		err = errors.Join(err, fields.ValidateTypeFields(t))
	}

	if _, ok := t.(appdef.IContainers); ok {
		err = errors.Join(err, containers.ValidateTypeContainers(t))
	}

	return err
}

// # Supports:
//   - appdef.IAppDefBuilder
type AppDefBuilder struct {
	comments.CommentBuilder
	app *AppDef
}

func NewAppDefBuilder(app *AppDef) *AppDefBuilder {
	return &AppDefBuilder{
		CommentBuilder: comments.MakeCommentBuilder(&app.WithComments),
		app:            app,
	}
}

func (ab *AppDefBuilder) AddPackage(localName, path string) appdef.IAppDefBuilder {
	ab.app.addPackage(localName, path)
	return ab
}

func (ab *AppDefBuilder) AddWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	return ab.app.addWorkspace(name)
}

func (ab *AppDefBuilder) AlterWorkspace(name appdef.QName) appdef.IWorkspaceBuilder {
	return ab.app.alterWorkspace(name)
}

func (ab AppDefBuilder) AppDef() appdef.IAppDef { return ab.app }

func (ab *AppDefBuilder) Build() (appdef.IAppDef, error) {
	if err := ab.app.build(); err != nil {
		return nil, err
	}
	return ab.app, nil
}

func (ab *AppDefBuilder) MustBuild() appdef.IAppDef {
	if err := ab.app.build(); err != nil {
		panic(err)
	}
	return ab.app
}

func (ab *AppDefBuilder) SetTypeComment(n appdef.QName, c ...string) {
	ab.app.setTypeComment(n, c...)
}
