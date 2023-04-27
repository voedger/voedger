/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type MockContainer struct {
	appdef.Container
	mock.Mock
}

func MockedContainer(name string, schema appdef.QName, min, max appdef.Occurs) *MockContainer {
	cnt := MockContainer{}
	cnt.
		On("Name").Return(name).
		On("Schema").Return(schema).
		On("MinOccurs").Return(min).
		On("MaxOccurs").Return(max)
	return &cnt
}

func (c *MockContainer) Name() string             { return c.Called().Get(0).(string) }
func (c *MockContainer) Schema() appdef.QName     { return c.Called().Get(0).(appdef.QName) }
func (c *MockContainer) MinOccurs() appdef.Occurs { return c.Called().Get(0).(appdef.Occurs) }
func (c *MockContainer) MaxOccurs() appdef.Occurs { return c.Called().Get(0).(appdef.Occurs) }
func (c *MockContainer) IsSys() bool              { return appdef.IsSysContainer(c.Name()) }
