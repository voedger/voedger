/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type Container struct {
	appdef.IContainer
	mock.Mock
}

func NewContainer(name string, def appdef.QName, min, max appdef.Occurs) *Container {
	cnt := Container{}
	cnt.
		On("Name").Return(name).
		On("Def").Return(def).
		On("MinOccurs").Return(min).
		On("MaxOccurs").Return(max)
	return &cnt
}

func (c *Container) Def() appdef.QName        { return c.Called().Get(0).(appdef.QName) }
func (c *Container) Name() string             { return c.Called().Get(0).(string) }
func (c *Container) MinOccurs() appdef.Occurs { return c.Called().Get(0).(appdef.Occurs) }
func (c *Container) MaxOccurs() appdef.Occurs { return c.Called().Get(0).(appdef.Occurs) }
func (c *Container) IsSys() bool              { return appdef.IsSysContainer(c.Name()) }
