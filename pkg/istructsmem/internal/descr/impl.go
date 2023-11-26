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
	}
	return &a
}

func (a *Application) read(app istructs.IAppStructs, rateLimits map[appdef.QName]map[istructs.RateLimitKind]istructs.RateLimit) {
	a.Packages = make(map[string]*Package)

	a.Name = app.AppQName()

	app.AppDef().Types(func(typ appdef.IType) {
		name := typ.QName()

		if name.Pkg() == appdef.SysPackage {
			return
		}

		pkg := getPkg(name, a)

		if data, ok := typ.(appdef.IData); ok {
			if !data.IsSystem() {
				d := newData()
				d.read(data)
				pkg.DataTypes[name.String()] = d
			}
			return
		}

		if str, ok := typ.(appdef.IStructure); ok {
			s := newStructure()
			s.read(str)
			pkg.Structures[name.String()] = s
			return
		}

		if view, ok := typ.(appdef.IView); ok {
			v := newView()
			v.read(view)
			pkg.Views[name.String()] = v
			return
		}

		if ext, ok := typ.(appdef.IExtension); ok {
			if pkg.Extensions == nil {
				pkg.Extensions = newExtensions()
			}
			pkg.Extensions.read(ext)
			return
		}
	})

	app.Resources().Resources(func(resName appdef.QName) {
		pkg := getPkg(resName, a)
		resource := newResource()
		resource.Name = resName
		pkg.Resources[resName.String()] = resource

		resource.read(app.Resources().QueryResource(resName))
	})

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
		Resources:  make(map[string]*Resource),
		RateLimits: make(map[string][]*RateLimit),
	}
}
