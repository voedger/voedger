/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type MockSchemaCache struct {
	appdef.SchemaCache
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

func (cache *MockSchemaCache) Schema(name appdef.QName) appdef.Schema {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			if s.QName() == name {
				return s
			}
		}
		return appdef.NullSchema
	}
	return cache.Called(name).Get(0).(appdef.Schema)
}

func (cache *MockSchemaCache) SchemaByName(name appdef.QName) appdef.Schema {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			if s.QName() == name {
				return s
			}
		}
		return nil
	}
	return cache.Called(name).Get(0).(appdef.Schema)
}

func (cache *MockSchemaCache) SchemaCount() int {
	if l := len(cache.sch); l > 0 {
		return l
	}
	return cache.Called().Get(0).(int)
}

func (cache *MockSchemaCache) Schemas(cb func(appdef.Schema)) {
	if len(cache.sch) > 0 {
		for _, s := range cache.sch {
			cb(s)
		}
		return
	}
	cache.Called(cb)
}
