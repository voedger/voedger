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

// # Implements:
//   - IUnique
type unique struct {
	owner  interface{}
	name   string
	fields []IField
	id     UniqueID
}

func newUnique(def interface{}, name string, fields []string) *unique {
	u := &unique{def, name, make([]IField, 0), NullUniqueID}
	sort.Strings(fields)
	fieldsDef := def.(IFields)
	for _, f := range fields {
		fld := fieldsDef.Field(f)
		if fld == nil {
			panic(fmt.Errorf("%v: can not create unique «%s»: field «%s» not found: %w", def.(IDef).QName(), name, f, ErrNameNotFound))
		}
		u.fields = append(u.fields, fld)
	}
	return u
}

func (u unique) Def() IDef {
	return u.owner.(IDef)
}

func (u unique) Name() string {
	return u.name
}

func (u unique) Fields() []IField {
	return u.fields
}

func (u unique) ID() UniqueID {
	return u.id
}

// Assigns ID. Must be called during application structures preparation
func (u *unique) SetID(value UniqueID) {
	u.id = value
}

// # Implements:
//   - IUniques
//   - IUniquesBuilder
type uniques struct {
	owner          interface{}
	uniques        map[string]*unique
	uniquesOrdered []string
	field          IField
}

func makeUniques(def interface{}) uniques {
	u := uniques{def, make(map[string]*unique), make([]string, 0), nil}
	return u
}

func (u *uniques) AddUnique(name string, fields []string) IUniquesBuilder {
	if name == NullName {
		name = generateUniqueName(u, fields)
	}
	return u.addUnique(name, fields)
}

func (u *uniques) SetUniqueField(name string) IUniquesBuilder {
	if name == NullName {
		u.field = nil
		return u
	}
	if ok, err := ValidIdent(name); !ok {
		panic((fmt.Errorf("%v: unique field name «%v» is invalid: %w", u.def().QName(), name, err)))
	}

	fld := u.fieldsDef().Field(name)
	if fld == nil {
		panic((fmt.Errorf("%v: unique field name «%v» not found: %w", u.def().QName(), name, ErrNameNotFound)))
	}
	if !fld.Required() {
		panic((fmt.Errorf("%v: unique field «%v» must be required", u.def().QName(), name)))
	}

	u.field = fld

	return u
}

func (u *uniques) UniqueByName(name string) IUnique {
	if u, ok := u.uniques[name]; ok {
		return u
	}
	return nil
}

func (u *uniques) UniqueByID(id UniqueID) (unique IUnique) {
	u.Uniques(func(u IUnique) {
		if u.ID() == id {
			unique = u
		}
	})
	return unique
}

func (u *uniques) UniqueCount() int {
	return len(u.uniques)
}

func (u *uniques) UniqueField() IField {
	return u.field
}

func (u *uniques) Uniques(enum func(IUnique)) {
	for _, n := range u.uniquesOrdered {
		enum(u.UniqueByName(n))
	}
}

func (u *uniques) addUnique(name string, fields []string) IUniquesBuilder {
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: unique name «%v» is invalid: %w", u.def().QName(), name, err))
	}
	if u.UniqueByName(name) != nil {
		panic(fmt.Errorf("%v: unique «%v» is already exists: %w", u.def().QName(), name, ErrNameUniqueViolation))
	}

	if len(fields) == 0 {
		panic(fmt.Errorf("%v: no fields specified for unique «%s»: %w", u.def().QName(), name, ErrNameMissed))
	}
	if i, j := duplicates(fields); i >= 0 {
		panic(fmt.Errorf("%v: unique «%s» has duplicates (fields[%d] == fields[%d] == %q): %w", u.def().QName(), name, i, j, fields[i], ErrNameUniqueViolation))
	}

	if len(fields) > MaxDefUniqueFieldsCount {
		panic(fmt.Errorf("%v: unique «%s» exceeds maximum fields (%d): %w", u.def().QName(), name, MaxDefUniqueFieldsCount, ErrTooManyFields))
	}

	u.Uniques(func(un IUnique) {
		ff := make([]string, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if overlaps(fields, ff) {
			panic(fmt.Errorf("%v: definition already has unique «%s» which overlaps with new unique: %w", u.def().QName(), name, ErrUniqueOverlaps))
		}
	})

	if len(u.uniques) >= MaxDefUniqueCount {
		panic(fmt.Errorf("%v: maximum uniques (%d) is exceeded: %w", u.def().QName(), MaxDefUniqueCount, ErrTooManyUniques))
	}

	un := newUnique(u.owner, name, fields)
	u.uniques[name] = un
	u.uniquesOrdered = append(u.uniquesOrdered, name)

	return u.owner.(IUniquesBuilder)
}

func (u *uniques) def() IDef {
	return u.owner.(IDef)
}

func (u *uniques) fieldsDef() IFields {
	return u.owner.(IFields)
}
