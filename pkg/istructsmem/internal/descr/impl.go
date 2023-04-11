/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

func newApplication() *Application {
	a := Application{
		Packages: make(map[string]*Package),
	}
	return &a
}

func (a *Application) read(app istructs.IAppStructs, rateLimits map[istructs.QName]map[istructs.RateLimitKind]istructs.RateLimit,
	uniquesByQNames map[istructs.QName][][]string) {
	a.Packages = make(map[string]*Package)

	a.Name = app.AppQName()

	app.Schemas().Schemas(func(schemaName istructs.QName) {
		pkg := getPkg(schemaName, a)
		schema := newSchema()
		schema.Name = schemaName
		pkg.Schemas[schemaName.String()] = schema
		schema.readAppSchema(app.Schemas().Schema(schemaName))
	})

	app.Resources().Resources(func(resName istructs.QName) {
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

	for qName, uniques := range uniquesByQNames {
		for _, fields := range uniques {
			pkg := getPkg(qName, a)
			unique := newUnique()
			unique.Fields = fields
			pkg.Uniques[qName.String()] = append(pkg.Uniques[qName.String()], unique)
		}
	}
}

func getPkg(schemaName istructs.QName, a *Application) *Package {
	pkgName := schemaName.Pkg()
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
		Schemas:    make(map[string]*Schema),
		Resources:  make(map[string]*Resource),
		RateLimits: make(map[string][]*RateLimit),
		Uniques:    make(map[string][]*Unique),
	}
}
