/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type Field struct {
	appdef.IField
	mock.Mock
	verify map[appdef.VerificationKind]bool
}

func NewField(name string, kind appdef.DataKind, req bool) *Field {
	fld := Field{}
	fld.
		On("Name").Return(name).
		On("DataKind").Return(kind).
		On("Required").Return(req).
		On("Verifiable").Return(false)
	return &fld
}

func NewVerifiedField(name string, kind appdef.DataKind, req bool, vk ...appdef.VerificationKind) *Field {
	fld := Field{verify: make(map[appdef.VerificationKind]bool)}
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

func (fld *Field) Name() string              { return fld.Called().Get(0).(string) }
func (fld *Field) DataKind() appdef.DataKind { return fld.Called().Get(0).(appdef.DataKind) }
func (fld *Field) Required() bool            { return fld.Called().Get(0).(bool) }
func (fld *Field) Verifiable() bool          { return fld.Called().Get(0).(bool) }
func (fld *Field) VerificationKind(vk appdef.VerificationKind) bool {
	if len(fld.verify) > 0 {
		return fld.verify[vk]
	}
	return fld.Called(vk).Get(0).(bool)
}
func (fld *Field) IsFixedWidth() bool { return fld.DataKind().IsFixed() }
func (fld *Field) IsSys() bool        { return appdef.IsSysField(fld.Name()) }
