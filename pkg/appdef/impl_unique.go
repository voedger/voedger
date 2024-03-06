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
	name   QName
	fields []IField
}

func newUnique(name QName, fieldNames []string, fields IFields) *unique {
	u := &unique{
		name:   name,
		fields: make([]IField, 0),
	}
	sort.Strings(fieldNames)
	for _, f := range fieldNames {
		fld := fields.Field(f)
		if fld == nil {
			panic(fmt.Errorf("can not create unique «%s»: field «%s» not found: %w", name, f, ErrNameNotFound))
		}
		u.fields = append(u.fields, fld)
	}
	return u
}

func (u unique) Name() QName {
	return u.name
}

func (u unique) Fields() []IField {
	return u.fields
}

// # Implements:
//   - IUniques
type uniques struct {
	app     *appDef
	fields  IFields
	uniques map[QName]IUnique
	field   IField
}

func makeUniques(app *appDef, fields IFields) uniques {
	uu := uniques{
		app:     app,
		fields:  fields,
		uniques: make(map[QName]IUnique),
	}
	return uu
}

func (uu *uniques) setUniqueField(name string) {
	if name == NullName {
		uu.field = nil
		return
	}
	if ok, err := ValidIdent(name); !ok {
		panic((fmt.Errorf("unique field name «%v» is invalid: %w", name, err)))
	}

	fld := uu.fields.Field(name)
	if fld == nil {
		panic((fmt.Errorf("unique field name «%v» not found: %w", name, ErrNameNotFound)))
	}

	uu.field = fld
}

func (uu uniques) UniqueByName(name QName) IUnique {
	if u, ok := uu.uniques[name]; ok {
		return u
	}
	return nil
}

func (uu uniques) UniqueCount() int {
	return len(uu.uniques)
}

func (uu uniques) UniqueField() IField {
	return uu.field
}

func (uu uniques) Uniques() map[QName]IUnique {
	return uu.uniques
}

func (uu *uniques) addUnique(name QName, fields []string, comment ...string) {
	if name == NullQName {
		panic(fmt.Errorf("unique name cannot be empty: %w", ErrNameMissed))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("unique name «%v» is invalid: %w", name, err))
	}
	if uu.UniqueByName(name) != nil {
		panic(fmt.Errorf("unique «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}

	if uu.app != nil {
		if t := uu.app.TypeByName(name); t != nil {
			panic(fmt.Errorf("unique name «%v» is already used by type %v: %w", name, t, ErrNameUniqueViolation))
		}
	}

	if len(fields) == 0 {
		panic(fmt.Errorf("no fields specified for unique «%v»: %w", name, ErrNameMissed))
	}
	if i, j := duplicates(fields); i >= 0 {
		panic(fmt.Errorf("unique «%v» has duplicates (fields[%d] == fields[%d] == %q): %w", name, i, j, fields[i], ErrNameUniqueViolation))
	}

	if len(fields) > MaxTypeUniqueFieldsCount {
		panic(fmt.Errorf("unique «%v» exceeds maximum fields (%d): %w", name, MaxTypeUniqueFieldsCount, ErrTooManyFields))
	}

	for n, un := range uu.uniques {
		ff := make([]string, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if overlaps(fields, ff) {
			panic(fmt.Errorf("type already has unique «%v» which overlaps with new unique «%v»: %w", n, name, ErrUniqueOverlaps))
		}
	}

	if len(uu.uniques) >= MaxTypeUniqueCount {
		panic(fmt.Errorf("maximum uniques (%d) is exceeded: %w", MaxTypeUniqueCount, ErrTooManyUniques))
	}

	un := newUnique(name, fields, uu.fields)
	un.comment.setComment(comment...)

	uu.uniques[name] = un
}

// # Implements:
//   - IUniquesBuilder
type uniquesBuilder struct {
	*uniques
}

func makeUniquesBuilder(uniques *uniques) uniquesBuilder {
	return uniquesBuilder{
		uniques: uniques,
	}
}

func (ub *uniquesBuilder) AddUnique(name QName, fields []string, comment ...string) IUniquesBuilder {
	ub.addUnique(name, fields, comment...)
	return ub
}

func (ub *uniquesBuilder) SetUniqueField(name string) IUniquesBuilder {
	ub.setUniqueField(name)
	return ub
}

// If the slices have duplicates, then the indices of the first pair are returned, otherwise (-1, -1)
func duplicates[T comparable](s []T) (int, int) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[i] == s[j] {
				return i, j
			}
		}
	}
	return -1, -1
}

// Returns is slice sub is a subset of slice set, i.e. all elements from sub exist in set
func subSet[T comparable](sub, set []T) bool {
	for _, v1 := range sub {
		found := false
		for _, v2 := range set {
			found = v1 == v2
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Returns is set1 and set2 overlaps, i.e. set1 is subset of set2 or set2 is subset of set1
func overlaps[T comparable](set1, set2 []T) bool {
	return subSet(set1, set2) || subSet(set2, set1)
}
