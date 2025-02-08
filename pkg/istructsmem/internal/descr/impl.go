/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newApplication() *Application {
	a := Application{
		Packages: make(map[string]*Package),
	}
	return &a
}

func (a *Application) read(name appdef.AppQName, app appdef.IAppDef) {
	a.Packages = make(map[string]*Package)

	for localName, fullPath := range app.Packages() {
		if localName == appdef.SysPackage {
			continue
		}
		pkg := newPackage()
		pkg.Name = localName
		pkg.Path = fullPath
		a.Packages[localName] = pkg
	}

	a.Name = name

	for _, ws := range app.Workspaces() {
		if ws.IsSystem() {
			continue
		}

		wsName := ws.QName()
		pkg := a.pkg(wsName.Pkg())

		w := newWorkspace()
		w.read(ws)

		pkg.Workspaces[wsName] = w
	}
}

func (a Application) pkg(name string) *Package {
	return a.Packages[name]
}

func newPackage() *Package {
	return &Package{
		Workspaces: make(map[appdef.QName]*Workspace),
	}
}
