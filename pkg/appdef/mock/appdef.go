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
	sch []*Schema
}

func NewAppDef(sch ...*Schema) *AppDef {
	app := AppDef{}
	if len(sch) > 0 {
		app.Add(sch...)
	}
	return &app
}

func (app *AppDef) Add(sch ...*Schema) {
	if len(sch) > 0 {
		for _, s := range sch {
			s.app = app
		}
		app.sch = append(app.sch, sch...)
	}
}

func (app *AppDef) Schema(name appdef.QName) appdef.Schema {
	if len(app.sch) > 0 {
		for _, s := range app.sch {
			if s.QName() == name {
				return s
			}
		}
		return appdef.NullSchema
	}
	return app.Called(name).Get(0).(appdef.Schema)
}

func (app *AppDef) SchemaByName(name appdef.QName) appdef.Schema {
	if len(app.sch) > 0 {
		for _, s := range app.sch {
			if s.QName() == name {
				return s
			}
		}
		return nil
	}
	return app.Called(name).Get(0).(appdef.Schema)
}

func (app *AppDef) SchemaCount() int {
	if l := len(app.sch); l > 0 {
		return l
	}
	return app.Called().Get(0).(int)
}

func (app *AppDef) Schemas(cb func(appdef.Schema)) {
	if len(app.sch) > 0 {
		for _, s := range app.sch {
			cb(s)
		}
		return
	}
	app.Called(cb)
}
