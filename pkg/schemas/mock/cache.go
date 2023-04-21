/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/schemas"
)

type MockSchemaCache struct {
	schemas.SchemaCache
	mock.Mock
	sch []*MockSchema
}

func MockedSchemaCache(sch ...*MockSchema) *MockSchemaCache {
	c := MockSchemaCache{}
	if len(sch) > 0 {
		c.MockSchemas(sch...)
	}
	return &c
}

func (cache *MockSchemaCache) MockSchemas(sch ...*MockSchema) {
	if len(sch) > 0 {
		for _, s := range sch {
			s.cache = cache
		}
		cache.sch = append(cache.sch, sch...)
	}
}

func (cache *MockSchemaCache) EnumSchemas(cb func(schemas.Schema)) {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			cb(s)
		}
		return
	}
	cache.Called(cb)
}

func (cache *MockSchemaCache) SchemaCount() int {
	if l := len(cache.sch); l > 0 {
		return l
	}
	return cache.Called().Get(0).(int)
}

func (cache *MockSchemaCache) SchemaByName(name schemas.QName) schemas.Schema {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			if s.QName() == name {
				return s
			}
		}
		return nil
	}
	return cache.Called(name).Get(0).(schemas.Schema)
}

func (cache *MockSchemaCache) Schema(name schemas.QName) schemas.Schema {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			if s.QName() == name {
				return s
			}
		}
		return schemas.NullSchema
	}
	return cache.Called(name).Get(0).(schemas.Schema)
}
