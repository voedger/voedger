/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"sort"
)

// # Implements:
//   - IUnique
type unique struct {
	comment
	emb    interface{}
	name   QName
	fields []IField
}

func newUnique(embeds interface{}, name QName, fields []string) *unique {
	u := &unique{
		emb:    embeds,
		name:   name,
		fields: make([]IField, 0),
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

func (u unique) Name() QName {
	return u.name
}

func (u unique) Fields() []IField {
	return u.fields
}

// # Implements:
//   - IUniques
//   - IUniquesBuilder
type uniques struct {
	emb     interface{}
	uniques map[QName]IUnique
	field   IField
}

func makeUniques(embeds interface{}) uniques {
	u := uniques{
		emb:     embeds,
		uniques: make(map[QName]IUnique),
	}
	return u
}

func (u *uniques) AddUnique(name QName, fields []string, comment ...string) IUniquesBuilder {
	return u.addUnique(name, fields, comment...)
}

func (u *uniques) SetUniqueField(name string) IUniquesBuilder {
	if name == NullName {
		u.field = nil
		return u
	}
	if ok, err := ValidIdent(name); !ok {
		panic((fmt.Errorf("%v: unique field name «%v» is invalid: %w", u.embeds(), name, err)))
	}

	fld := u.embeds().Field(name)
	if fld == nil {
		panic((fmt.Errorf("%v: unique field name «%v» not found: %w", u.embeds(), name, ErrNameNotFound)))
	}

	u.field = fld

	return u
}

func (u *uniques) UniqueByName(name QName) IUnique {
	if u, ok := u.uniques[name]; ok {
		return u
	}
	return nil
}

func (u *uniques) UniqueCount() int {
	return len(u.uniques)
}

func (u *uniques) UniqueField() IField {
	return u.field
}

func (u *uniques) Uniques() map[QName]IUnique {
	return u.uniques
}

func (u *uniques) addUnique(name QName, fields []string, comment ...string) IUniquesBuilder {
	if name == NullQName {
		panic(fmt.Errorf("%v: unique name cannot be empty: %w", u.embeds(), ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("%v: unique name «%v» is invalid: %w", u.embeds(), name, err))
	}
	if u.UniqueByName(name) != nil {
		panic(fmt.Errorf("%v: unique «%v» is already exists: %w", u.embeds(), name, ErrNameUniqueViolation))
	}

	if app := u.embeds().App(); app != nil {
		if t := app.TypeByName(name); t != nil {
			panic(fmt.Errorf("%v: unique name «%v» is already used by type %v: %w", u.embeds(), name, t, ErrNameUniqueViolation))
		}
	}

	if len(fields) == 0 {
		panic(fmt.Errorf("%v: no fields specified for unique «%v»: %w", u.embeds(), name, ErrNameMissed))
	}
	if i, j := duplicates(fields); i >= 0 {
		panic(fmt.Errorf("%v: unique «%v» has duplicates (fields[%d] == fields[%d] == %q): %w", u.embeds(), name, i, j, fields[i], ErrNameUniqueViolation))
	}

	if len(fields) > MaxTypeUniqueFieldsCount {
		panic(fmt.Errorf("%v: unique «%v» exceeds maximum fields (%d): %w", u.embeds(), name, MaxTypeUniqueFieldsCount, ErrTooManyFields))
	}

	for n, un := range u.uniques {
		ff := make([]string, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if overlaps(fields, ff) {
			panic(fmt.Errorf("%v: type already has unique «%v» which overlaps with new unique «%v»: %w", u.embeds(), n, name, ErrUniqueOverlaps))
		}
	}

	if len(u.uniques) >= MaxTypeUniqueCount {
		panic(fmt.Errorf("%v: maximum uniques (%d) is exceeded: %w", u.embeds(), MaxTypeUniqueCount, ErrTooManyUniques))
	}

	un := newUnique(u.emb, name, fields)
	un.SetComment(comment...)

	u.uniques[name] = un

	return u.emb.(IUniquesBuilder)
}

func (u *uniques) embeds() IStructure {
	return u.emb.(IStructure)
}
