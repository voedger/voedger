/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apps

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/acl"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
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
	types.WithTypes
}

func NewAppDef() *AppDef {
	app := AppDef{
		WithComments:   comments.MakeWithComments(),
		WithPackages:   packages.MakeWithPackages(),
		WithWorkspaces: workspaces.MakeWithWorkspaces(),
		WithACL:        acl.MakeWithACL(),
		WithTypes:      types.MakeWithTypes(),
	}
	return &app
}

func (app *AppDef) AppendType(t appdef.IType) {
	app.WithTypes.AppendType(t)
	app.changed()
}

func (app *AppDef) build() (err error) {
	err = app.Build()
	if err == nil {
		app.Builded()
	}
	return err
}

func (app *AppDef) changed() { app.Changed() }

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
