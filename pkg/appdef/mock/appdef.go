/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type AppDef struct {
	appdef.IAppDef
	mock.Mock
	defs []*Def
}

func NewAppDef(def ...*Def) *AppDef {
	app := AppDef{}
	if len(def) > 0 {
		app.AddStruct(def...)
	}
	return &app
}

func (app *AppDef) AddStruct(def ...*Def) {
	if len(def) > 0 {
		for _, d := range def {
			d.app = app
		}
		app.defs = append(app.defs, def...)
	}
}

func (app *AppDef) AddView(view *View) {
	view.app = app
	app.AddStruct(
		view.view,
		view.pk,
		view.cc,
		view.val,
	)
}

func (app *AppDef) Def(name appdef.QName) appdef.IDef {
	if len(app.defs) > 0 {
		for _, d := range app.defs {
			if d.QName() == name {
				return d
			}
		}
		return appdef.NullDef
	}
	return app.Called(name).Get(0).(appdef.IDef)
}

func (app *AppDef) DefByName(name appdef.QName) appdef.IDef {
	if len(app.defs) > 0 {
		for _, d := range app.defs {
			if d.QName() == name {
				return d
			}
		}
		return nil
	}
	return app.Called(name).Get(0).(appdef.IDef)
}

func (app *AppDef) DefCount() int {
	if l := len(app.defs); l > 0 {
		return l
	}
	return app.Called().Get(0).(int)
}

func (app *AppDef) Defs(cb func(appdef.IDef)) {
	if len(app.defs) > 0 {
		for _, d := range app.defs {
			cb(d)
		}
		return
	}
	app.Called(cb)
}
