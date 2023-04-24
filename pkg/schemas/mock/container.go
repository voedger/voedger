/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/schemas"
)

type MockContainer struct {
	schemas.Container
	mock.Mock
}

func MockedContainer(name string, schema schemas.QName, min, max schemas.Occurs) *MockContainer {
	cnt := MockContainer{}
	cnt.
		On("Name").Return(name).
		On("Schema").Return(schema).
		On("MinOccurs").Return(min).
		On("MaxOccurs").Return(max)
	return &cnt
}

func (c *MockContainer) Name() string              { return c.Called().Get(0).(string) }
func (c *MockContainer) Schema() schemas.QName     { return c.Called().Get(0).(schemas.QName) }
func (c *MockContainer) MinOccurs() schemas.Occurs { return c.Called().Get(0).(schemas.Occurs) }
func (c *MockContainer) MaxOccurs() schemas.Occurs { return c.Called().Get(0).(schemas.Occurs) }
func (c *MockContainer) IsSys() bool               { return schemas.IsSysContainer(c.Name()) }
