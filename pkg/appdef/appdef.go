/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// Implements IAppDef and IAppDefBuilder interfaces
type appDef struct {
	changes int
	defs    map[QName]*def
}

func newAppDef() *appDef {
	app := appDef{
		defs: make(map[QName]*def),
	}
	return &app
}

func (app *appDef) Add(name QName, kind DefKind) IDefBuilder {
	if name == NullQName {
		panic(fmt.Errorf("definition name cannot be empty: %w", ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("invalid definition name «%v»: %w", name, err))
	}
	if app.DefByName(name) != nil {
		panic(fmt.Errorf("definition name «%s» already used: %w", name, ErrNameUniqueViolation))
	}
	d := newDef(app, name, kind)
	app.defs[name] = d
	app.changed()
	return d
}

func (app *appDef) AddView(name QName) IViewBuilder {
	v := newViewBuilder(app, name)
	app.changed()
	return &v
}

func (app *appDef) Build() (result IAppDef, err error) {
	app.prepare()

	validator := newValidator()
	app.Defs(func(d IDef) {
		err = errors.Join(err, validator.validate(d))
	})
	if err != nil {
		return nil, err
	}

	app.changes = 0
	return app, nil
}

func (app *appDef) HasChanges() bool {
	return app.changes > 0
}

func (app *appDef) Def(name QName) IDef {
	if d := app.DefByName(name); d != nil {
		return d
	}
	return NullDef
}

func (app *appDef) DefByName(name QName) IDef {
	if d, ok := app.defs[name]; ok {
		return d
	}
	return nil
}

func (app *appDef) DefCount() int {
	return len(app.defs)
}

func (app *appDef) Defs(cb func(IDef)) {
	for _, d := range app.defs {
		cb(d)
	}
}

func (app *appDef) changed() {
	app.changes++
}

func (app *appDef) prepare() {
	app.Defs(func(d IDef) {
		if d.Kind() == DefKind_ViewRecord {
			app.prepareViewFullKeyDef(d)
		}
	})
}

func (app *appDef) remove(name QName) {
	delete(app.defs, name)
	app.changed()
}
