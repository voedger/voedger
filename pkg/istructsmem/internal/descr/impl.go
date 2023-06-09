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

	app.AppDef().Defs(func(def appdef.IDef) {
		defName := def.QName()
		pkg := getPkg(defName, a)
		d := newDef()
		d.Name = defName
		pkg.Defs[defName.String()] = d
		d.read(app.AppDef().Def(defName))
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
		Defs:       make(map[string]*Def),
		Resources:  make(map[string]*Resource),
		RateLimits: make(map[string][]*RateLimit),
	}
}
