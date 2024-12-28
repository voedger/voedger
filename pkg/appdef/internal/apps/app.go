/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apps

import (
	"errors"
	"iter"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/acl"
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
	packages.WithPackages
	workspaces.WithWorkspaces
	acl.WithACL
	sysWS *workspaces.Workspace
	types *types.Types[appdef.IType]
}

func NewAppDef() *AppDef {
	app := AppDef{
		WithComments:   comments.MakeWithComments(),
		WithPackages:   packages.MakeWithPackages(),
		WithWorkspaces: workspaces.MakeWithWorkspaces(),
		WithACL:        acl.MakeWithACL(),
		types:          types.NewTypes[appdef.IType](),
	}
	app.makeSysPackage()
	return &app
}

func (app *AppDef) AppendType(t appdef.IType) {
	name := t.QName()
	if name == appdef.NullQName {
		panic(appdef.ErrMissed("%s type name", t.Kind().TrimString()))
	}
	app.types.Add(t)
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
	packages.AddPackage(&app.WithPackages, appdef.SysPackage, appdef.SysPackagePath)
	app.makeSysWorkspace()
}

// Makes system workspace.
func (app *AppDef) makeSysWorkspace() {
	app.sysWS = workspaces.AddWorkspace(app, &app.WithWorkspaces, appdef.SysWorkspaceQName)
	app.AppendType(app.sysWS)

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
	packages.PackagesBuilder
	workspaces.WorkspacesBuilder
	app *AppDef
}

func NewAppDefBuilder(app *AppDef) *AppDefBuilder {
	return &AppDefBuilder{
		CommentBuilder:    comments.MakeCommentBuilder(&app.WithComments),
		PackagesBuilder:   packages.MakePackagesBuilder(&app.WithPackages),
		WorkspacesBuilder: workspaces.MakeWorkspacesBuilder(app, &app.WithWorkspaces),
		app:               app,
	}
}

func (ab AppDefBuilder) AppDef() appdef.IAppDef { return ab.app }

func (ab *AppDefBuilder) Build() (appdef.IAppDef, error) {
	if err := ab.app.build(); err != nil {
		return nil, err
	}
	return ab.app, nil
}

func (ab *AppDefBuilder) MustBuild() appdef.IAppDef {
	a, err := ab.Build()
	if err != nil {
		panic(err)
	}
	return a
}

func (ab *AppDefBuilder) SetTypeComment(n appdef.QName, c ...string) {
	ab.app.setTypeComment(n, c...)
}
