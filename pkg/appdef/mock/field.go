/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type MockField struct {
	appdef.Field
	mock.Mock
	verify map[appdef.VerificationKind]bool
}

func MockedField(name string, kind appdef.DataKind, req bool) *MockField {
	fld := MockField{}
	fld.
		On("Name").Return(name).
		On("DataKind").Return(kind).
		On("Required").Return(req).
		On("Verifiable").Return(false)
	return &fld
}

func MockedVerifiedField(name string, kind appdef.DataKind, req bool, vk ...appdef.VerificationKind) *MockField {
	fld := MockField{verify: make(map[appdef.VerificationKind]bool)}
	for _, k := range vk {
		fld.verify[k] = true
	}
	fld.
		On("Name").Return(name).
		On("DataKind").Return(kind).
		On("Required").Return(req).
		On("Verifiable").Return(true)
	return &fld
}

func (fld *MockField) Name() string              { return fld.Called().Get(0).(string) }
func (fld *MockField) DataKind() appdef.DataKind { return fld.Called().Get(0).(appdef.DataKind) }
func (fld *MockField) Required() bool            { return fld.Called().Get(0).(bool) }
func (fld *MockField) Verifiable() bool          { return fld.Called().Get(0).(bool) }
func (fld *MockField) VerificationKind(vk appdef.VerificationKind) bool {
	if len(fld.verify) > 0 {
		return fld.verify[vk]
	}
	return fld.Called(vk).Get(0).(bool)
}
func (fld *MockField) IsFixedWidth() bool { return fld.DataKind().IsFixed() }
func (fld *MockField) IsSys() bool        { return appdef.IsSysField(fld.Name()) }
