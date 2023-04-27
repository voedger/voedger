/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type MockSchema struct {
	appdef.Schema
	mock.Mock
	cache      *MockSchemaCache
	fields     []*MockField
	containers []*MockContainer
}

func MockedSchema(name appdef.QName, kind appdef.SchemaKind, fields ...*MockField) *MockSchema {
	s := MockSchema{}
	s.
		On("QName").Return(name).
		On("Kind").Return(kind)
	if len(fields) > 0 {
		s.MockFields(fields...)
	}
	return &s
}

func (s *MockSchema) MockFields(f ...*MockField) {
	s.fields = append(s.fields, f...)
}

func (s *MockSchema) MockContainers(c ...*MockContainer) {
	s.containers = append(s.containers, c...)
}

func (s *MockSchema) Cache() appdef.SchemaCache {
	if s.cache != nil {
		return s.cache
	}
	return s.Called().Get(0).(appdef.SchemaCache)
}

func (s *MockSchema) QName() appdef.QName     { return s.Called().Get(0).(appdef.QName) }
func (s *MockSchema) Kind() appdef.SchemaKind { return s.Called().Get(0).(appdef.SchemaKind) }

func (s *MockSchema) Field(name string) appdef.Field {
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

func (s *MockSchema) FieldCount() int {
	if l := len(s.fields); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *MockSchema) Fields(cb func(appdef.Field)) {
	if len(s.fields) > 0 {
		for _, f := range s.fields {
			cb(f)
		}
		return
	}
	s.Called(cb)
}

func (s *MockSchema) Container(name string) appdef.Container {
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

func (s *MockSchema) ContainerCount() int {
	if l := len(s.containers); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *MockSchema) Containers(cb func(appdef.Container)) {
	if len(s.containers) > 0 {
		for _, c := range s.containers {
			cb(c)
		}
		return
	}
	s.Called(cb)
}

func (s *MockSchema) ContainerSchema(name string) appdef.Schema {
	if (s.cache != nil) && (len(s.containers) > 0) {
		if c := s.Container(name); c != nil {
			return s.cache.Schema(c.Schema())
		}
		return appdef.NullSchema
	}
	return s.Called(name).Get(0).(appdef.Schema)
}

func (s *MockSchema) Singleton() bool { return s.Called().Get(0).(bool) }
func (s *MockSchema) Validate() error { return s.Called().Get(0).(error) }
