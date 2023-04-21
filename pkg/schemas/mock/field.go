/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/schemas"
)

type MockField struct {
	schemas.Field
	mock.Mock
}

func MockedField(name string, kind schemas.DataKind, req bool) *MockField {
	fld := MockField{}
	fld.
		On("Name").Return(name).
		On("Kind").Return(kind).
		On("Required").Return(req)
	return &fld
}

func (fld *MockField) Name() string               { return fld.Called().Get(0).(string) }
func (fld *MockField) DataKind() schemas.DataKind { return fld.Called().Get(0).(schemas.DataKind) }
func (fld *MockField) Required() bool             { return fld.Called().Get(0).(bool) }
func (fld *MockField) Verifiable() bool           { return fld.Called().Get(0).(bool) }
func (fld *MockField) IsFixedWidth() bool         { return fld.DataKind().IsFixed() }
func (fld *MockField) IsSys() bool                { return schemas.IsSysField(fld.Name()) }
