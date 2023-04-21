/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/schemas"
)

type MockSchema struct {
	schemas.Schema
	mock.Mock
	cache      *MockSchemaCache
	fields     []*MockField
	containers []*MockContainer
}

func MockedSchema(name schemas.QName, kind schemas.SchemaKind, fields ...*MockField) *MockSchema {
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

func (s *MockSchema) Cache() schemas.SchemaCache {
	if s.cache != nil {
		return s.cache
	}
	return s.Called().Get(0).(schemas.SchemaCache)
}

func (s *MockSchema) QName() schemas.QName     { return s.Called().Get(0).(schemas.QName) }
func (s *MockSchema) Kind() schemas.SchemaKind { return s.Called().Get(0).(schemas.SchemaKind) }

func (s *MockSchema) Field(name string) schemas.Field {
	if len(s.fields) > 0 {
		for _, f := range s.fields {
			if f.Name() == name {
				return f
			}
		}
		return nil
	}
	return s.Called(name).Get(0).(schemas.Field)
}

func (s *MockSchema) FieldCount() int {
	if l := len(s.fields); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *MockSchema) Fields(cb func(schemas.Field)) {
	if len(s.fields) > 0 {
		for _, f := range s.fields {
			cb(f)
		}
		return
	}
	s.Called(cb)
}

func (s *MockSchema) Container(name string) schemas.Container {
	if len(s.containers) > 0 {
		for _, c := range s.containers {
			if c.Name() == name {
				return c
			}
		}
		return nil
	}
	return s.Called(name).Get(0).(schemas.Container)
}

func (s *MockSchema) ContainerCount() int {
	if l := len(s.containers); l > 0 {
		return l
	}
	return s.Called().Get(0).(int)
}

func (s *MockSchema) Containers(cb func(schemas.Container)) {
	if len(s.containers) > 0 {
		for _, c := range s.containers {
			cb(c)
		}
		return
	}
	s.Called(cb)
}

func (s *MockSchema) ContainerSchema(name string) schemas.Schema {
	if (s.cache != nil) && (len(s.containers) > 0) {
		if c := s.Container(name); c != nil {
			return s.cache.Schema(c.Schema())
		}
		return schemas.NullSchema
	}
	return s.Called(name).Get(0).(schemas.Schema)
}

func (s *MockSchema) Singleton() bool { return s.Called().Get(0).(bool) }
func (s *MockSchema) Validate() error { return s.Called().Get(0).(error) }
