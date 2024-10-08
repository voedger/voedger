/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func newApplication() *Application {
	a := Application{
		Packages: make(map[string]*Package),
		ACL:      newACL(),
	}
	return &a
}

func (a *Application) read(app istructs.IAppStructs, rateLimits map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit) {
	a.Packages = make(map[string]*Package)

	for localName, fullPath := range app.AppDef().Packages {
		if localName == appdef.SysPackage {
			continue
		}
		pkg := newPackage()
		pkg.Name = localName
		pkg.Path = fullPath
		a.Packages[localName] = pkg
	}

	a.Name = app.AppQName()

	for typ := range app.AppDef().Types {
		name := typ.QName()

		if name.Pkg() == appdef.SysPackage {
			continue
		}

		pkg := getPkg(name, a)
		switch t := typ.(type) {
		case appdef.IData:
			if !t.IsSystem() {
				d := newData()
				d.read(t)
				pkg.DataTypes[name.String()] = d
			}
		case appdef.IStructure:
			s := newStructure()
			s.read(t)
			pkg.Structures[name.String()] = s
		case appdef.IView:
			v := newView()
			v.read(t)
			pkg.Views[name.String()] = v
		case appdef.IExtension:
			if pkg.Extensions == nil {
				pkg.Extensions = newExtensions()
			}
			pkg.Extensions.read(t)
		case appdef.IRole:
			r := newRole()
			r.read(t)
			pkg.Roles[name.String()] = r
		case appdef.IWorkspace:
			w := newWorkspace()
			w.read(t)
			pkg.Workspaces[name.String()] = w
		}
	}

	a.ACL.read(app.AppDef(), true)

	for qName, qNameRateLimit := range rateLimits {
		pkg := getPkg(qName, a)
		for rlKind, rl := range qNameRateLimit {
			rateLimit := newRateLimit()
			rateLimit.Kind = rlKind
			rateLimit.MaxAllowedPerDuration = rl.MaxAllowedPerDuration
			rateLimit.Period = rl.Period
			pkg.RateLimits[qName.String()] = append(pkg.RateLimits[qName.String()], rateLimit)
		}
	}
}

func getPkg(name appdef.QName, a *Application) *Package {
	pkgName := name.Pkg()
	pkg := a.Packages[pkgName]
	if pkg == nil {
		pkg = newPackage()
		pkg.Name = pkgName
		a.Packages[pkgName] = pkg
	}
	return pkg
}

func newPackage() *Package {
	return &Package{
		DataTypes:  make(map[string]*Data),
		Structures: make(map[string]*Structure),
		Views:      make(map[string]*View),
		Roles:      make(map[string]*Role),
		Workspaces: make(map[string]*Workspace),
		Resources:  make(map[string]*Resource),
		RateLimits: make(map[string][]*RateLimit),
	}
}
