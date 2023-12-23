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
	comment
	emb    interface{}
	name   string
	fields []IField
	id     UniqueID
}

func newUnique(embeds interface{}, name string, fields []string) *unique {
	u := &unique{
		emb:    embeds,
		name:   name,
		fields: make([]IField, 0),
		id:     NullUniqueID,
	}
	sort.Strings(fields)
	str := embeds.(IStructure)
	for _, f := range fields {
		fld := str.Field(f)
		if fld == nil {
			panic(fmt.Errorf("%v: can not create unique «%s»: field «%s» not found: %w", str, name, f, ErrNameNotFound))
		}
		u.fields = append(u.fields, fld)
	}
	return u
}

func (u unique) ParentStructure() IStructure {
	return u.emb.(IStructure)
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
	emb            interface{}
	uniques        map[string]*unique
	uniquesOrdered []IUnique
}

func makeUniques(embeds interface{}) uniques {
	u := uniques{embeds, make(map[string]*unique), make([]IUnique, 0)}
	return u
}

func (u *uniques) AddUnique(name string, fields []string, comment ...string) IUniquesBuilder {
	if name == NullName {
		name = generateUniqueName(u, fields)
	}
	return u.addUnique(name, fields, comment...)
}

func (u *uniques) UniqueByName(name string) IUnique {
	if u, ok := u.uniques[name]; ok {
		return u
	}
	return nil
}

func (u *uniques) UniqueByID(id UniqueID) IUnique {
	for _, u := range u.uniquesOrdered {
		if u.ID() == id {
			return u
		}
	}
	return nil
}

func (u *uniques) UniqueCount() int {
	return len(u.uniques)
}

func (u *uniques) Uniques() []IUnique {
	return u.uniquesOrdered
}

func (u *uniques) addUnique(name string, fields []string, comment ...string) IUniquesBuilder {
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: unique name «%v» is invalid: %w", u.embeds(), name, err))
	}
	if u.UniqueByName(name) != nil {
		panic(fmt.Errorf("%v: unique «%v» is already exists: %w", u.embeds(), name, ErrNameUniqueViolation))
	}

	if len(fields) == 0 {
		panic(fmt.Errorf("%v: no fields specified for unique «%s»: %w", u.embeds(), name, ErrNameMissed))
	}
	if i, j := duplicates(fields); i >= 0 {
		panic(fmt.Errorf("%v: unique «%s» has duplicates (fields[%d] == fields[%d] == %q): %w", u.embeds(), name, i, j, fields[i], ErrNameUniqueViolation))
	}

	if len(fields) > MaxTypeUniqueFieldsCount {
		panic(fmt.Errorf("%v: unique «%s» exceeds maximum fields (%d): %w", u.embeds(), name, MaxTypeUniqueFieldsCount, ErrTooManyFields))
	}

	for _, un := range u.uniquesOrdered {
		ff := make([]string, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if overlaps(fields, ff) {
			panic(fmt.Errorf("%v: type already has unique «%s» which overlaps with new unique: %w", u.embeds(), name, ErrUniqueOverlaps))
		}
	}

	if len(u.uniques) >= MaxTypeUniqueCount {
		panic(fmt.Errorf("%v: maximum uniques (%d) is exceeded: %w", u.embeds(), MaxTypeUniqueCount, ErrTooManyUniques))
	}

	un := newUnique(u.emb, name, fields)
	un.SetComment(comment...)
	u.uniques[name] = un
	u.uniquesOrdered = append(u.uniquesOrdered, un)

	return u.emb.(IUniquesBuilder)
}

func (u *uniques) embeds() IStructure {
	return u.emb.(IStructure)
}
