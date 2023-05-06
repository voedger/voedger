/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"sort"
)

const (
	NullUniqueID  UniqueID = 0
	FirstUniqueID          = NullUniqueID + 65536
)

// Implements IUnique interface
type unique struct {
	def    *def
	name   string
	fields []IField
	id     UniqueID
}

func newUnique(def *def, name string, fields []string) *unique {
	u := unique{def, name, make([]IField, 0), NullUniqueID}
	sort.Strings(fields)
	for _, f := range fields {
		fld := def.Field(f)
		if fld == nil {
			panic(fmt.Errorf("%v: can not create unique «%s»: field «%s» not found: %w", def.QName(), name, f, ErrNameNotFound))
		}
		u.fields = append(u.fields, fld)
	}
	return &u
}

func (u unique) Def() IDef { return u.def }

func (u unique) Name() string { return u.name }

func (u unique) Fields() []IField { return u.fields }

func (u unique) ID() UniqueID { return u.id }

// Assigns ID. Must be called during application structures preparation
func (u *unique) SetID(value UniqueID) { u.id = value }
