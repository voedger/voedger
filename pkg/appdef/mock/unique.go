/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package mock

import (
	"sort"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
)

type Unique struct {
	appdef.IUnique
	mock.Mock
	def    *Def
	fields []string
}

func NewUnique(name string, fields []string) *Unique {
	sort.Strings(fields)
	u := Unique{fields: fields}
	u.
		On("Name").Return(name)
	return &u
}

func (u *Unique) Def() appdef.IDef {
	if u.def != nil {
		return u.def
	}
	return u.Called().Get(0).(appdef.IDef)
}

func (u *Unique) Name() string { return u.Called().Get(0).(string) }

func (u *Unique) Fields() (fields []appdef.IField) {
	if (u.def != nil) && (len(u.fields) > 0) {
		for _, n := range u.fields {
			f := u.def.Field(n)
			fields = append(fields, f)
		}
		return fields
	}
	return u.Called().Get(0).([]appdef.IField)
}

func (u *Unique) ID() appdef.UniqueID      { return u.Called().Get(0).(appdef.UniqueID) }
func (u *Unique) SetID(id appdef.UniqueID) { u.On("ID").Return(id) }
