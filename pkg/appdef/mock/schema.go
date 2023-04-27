/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type Schema struct {
	appdef.Schema
	mock.Mock
	app        *AppDef
	fields     []*Field
	containers []*Container
}

func NewSchema(name appdef.QName, kind appdef.SchemaKind, fields ...*Field) *Schema {
	s := Schema{}
	s.
		On("QName").Return(name).
		On("Kind").Return(kind)
	if len(fields) > 0 {
		s.AddField(fields...)
	}
	return &s
}

func (s *Schema) AddField(f ...*Field) {
	s.fields = append(s.fields, f...)
}

func (s *Schema) AddContainer(c ...*Container) {
	s.containers = append(s.containers, c...)
}

func (s *Schema) App() appdef.IAppDef {
	if s.app != nil {
		return s.app
	}
	return s.Called().Get(0).(appdef.IAppDef)
}

func (s *Schema) QName() appdef.QName     { return s.Called().Get(0).(appdef.QName) }
func (s *Schema) Kind() appdef.SchemaKind { return s.Called().Get(0).(appdef.SchemaKind) }

func (s *Schema) Field(name string) appdef.Field {
	if len(s.fields) > 0 {
		for _, f := range s.fields {
			if f.Name() == name {
				return f
			}
		}
		return nil
	}
	return s.Called(name).Get(0).(appdef.Field)
}

func (s *Schema) FieldCount() int {
	if l := len(s.fields); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *Schema) Fields(cb func(appdef.Field)) {
	if len(s.fields) > 0 {
		for _, f := range s.fields {
			cb(f)
		}
		return
	}
	s.Called(cb)
}

func (s *Schema) Container(name string) appdef.Container {
	if len(s.containers) > 0 {
		for _, c := range s.containers {
			if c.Name() == name {
				return c
			}
		}
		return nil
	}
	return s.Called(name).Get(0).(appdef.Container)
}

func (s *Schema) ContainerCount() int {
	if l := len(s.containers); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *Schema) Containers(cb func(appdef.Container)) {
	if len(s.containers) > 0 {
		for _, c := range s.containers {
			cb(c)
		}
		return
	}
	s.Called(cb)
}

func (s *Schema) ContainerSchema(name string) appdef.Schema {
	if (s.app != nil) && (len(s.containers) > 0) {
		if c := s.Container(name); c != nil {
			return s.app.Schema(c.Schema())
		}
		return appdef.NullSchema
	}
	return s.Called(name).Get(0).(appdef.Schema)
}

func (s *Schema) Singleton() bool { return s.Called().Get(0).(bool) }
func (s *Schema) Validate() error { return s.Called().Get(0).(error) }
