/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type Def struct {
	appdef.IDef
	mock.Mock
	app        *AppDef
	fields     []*Field
	containers []*Container
}

func NewDef(name appdef.QName, kind appdef.DefKind, fields ...*Field) *Def {
	def := Def{}
	def.
		On("QName").Return(name).
		On("Kind").Return(kind)
	if len(fields) > 0 {
		def.AddField(fields...)
	}
	return &def
}

func (d *Def) AddField(f ...*Field) {
	d.fields = append(d.fields, f...)
}

func (d *Def) AddContainer(c ...*Container) {
	d.containers = append(d.containers, c...)
}

func (d *Def) App() appdef.IAppDef {
	if d.app != nil {
		return d.app
	}
	return d.Called().Get(0).(appdef.IAppDef)
}

func (d *Def) QName() appdef.QName  { return d.Called().Get(0).(appdef.QName) }
func (d *Def) Kind() appdef.DefKind { return d.Called().Get(0).(appdef.DefKind) }

func (d *Def) Field(name string) appdef.Field {
	if len(d.fields) > 0 {
		for _, f := range d.fields {
			if f.Name() == name {
				return f
			}
		}
		return nil
	}
	return d.Called(name).Get(0).(appdef.Field)
}

func (d *Def) FieldCount() int {
	if l := len(d.fields); l > 0 {
		return l
	}
	return d.Called().Get(0).(int)
}

func (d *Def) Fields(cb func(appdef.Field)) {
	if len(d.fields) > 0 {
		for _, f := range d.fields {
			cb(f)
		}
		return
	}
	d.Called(cb)
}

func (d *Def) Container(name string) appdef.Container {
	if len(d.containers) > 0 {
		for _, c := range d.containers {
			if c.Name() == name {
				return c
			}
		}
		return nil
	}
	return d.Called(name).Get(0).(appdef.Container)
}

func (d *Def) ContainerCount() int {
	if l := len(d.containers); l > 0 {
		return l
	}
	return d.Called().Get(0).(int)
}

func (d *Def) Containers(cb func(appdef.Container)) {
	if len(d.containers) > 0 {
		for _, c := range d.containers {
			cb(c)
		}
		return
	}
	d.Called(cb)
}

func (d *Def) ContainerDef(name string) appdef.IDef {
	if (d.app != nil) && (len(d.containers) > 0) {
		if c := d.Container(name); c != nil {
			return d.app.Def(c.Def())
		}
		return appdef.NullDef
	}
	return d.Called(name).Get(0).(appdef.IDef)
}

func (d *Def) Singleton() bool { return d.Called().Get(0).(bool) }
