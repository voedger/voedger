/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef/internal/slicex"
)

// # Implements:
//   - IUnique
type unique struct {
	comment
	name   QName
	fields []IField
}

func newUnique(name QName, fieldNames []FieldName, fields IFields) *unique {
	u := &unique{
		name:   name,
		fields: make([]IField, 0),
	}
	slices.Sort(fieldNames)
	for _, f := range fieldNames {
		fld := fields.Field(f)
		if fld == nil {
			panic(ErrFieldNotFound(f))
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

func (u unique) String() string {
	return fmt.Sprintf("unique «%v»", u.name)
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

func (uu *uniques) setUniqueField(name FieldName) {
	if name == NullName {
		uu.field = nil
		return
	}
	if ok, err := ValidFieldName(name); !ok {
		panic(fmt.Errorf("unique field name «%v» is invalid: %w", name, err))
	}

	fld := uu.fields.Field(name)
	if fld == nil {
		panic(ErrFieldNotFound(name))
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

func (uu *uniques) addUnique(name QName, fields []FieldName, comment ...string) {
	if name == NullQName {
		panic(ErrMissed("unique name"))
	}
	if ok, err := ValidQName(name); !ok {
		panic(fmt.Errorf("unique name «%v» is invalid: %w", name, err))
	}
	if uu.UniqueByName(name) != nil {
		panic(ErrAlreadyExists("unique «%v»", name))
	}

	if uu.app != nil {
		if t := uu.app.Type(name); t.Kind() != TypeKind_null {
			panic(ErrAlreadyExists("name «%v» already used for %v", name, t))
		}
	}

	if len(fields) == 0 {
		panic(ErrMissed("unique «%v» fields", name))
	}
	if i, j := slicex.Duplicates(fields); i >= 0 {
		panic(ErrAlreadyExists("fields in unique «%v» has duplicates (fields[%d] == fields[%d] == %q)", name, i, j, fields[i]))
	}

	if len(fields) > MaxTypeUniqueFieldsCount {
		panic(ErrTooMany("fields in unique «%v», maximum is %d", name, MaxTypeUniqueFieldsCount))
	}

	for _, un := range uu.uniques {
		ff := make([]FieldName, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if slicex.Overlaps(fields, ff) {
			panic(ErrAlreadyExists("type already has %v which fields overlaps new unique fields", un))
		}
	}

	if len(uu.uniques) >= MaxTypeUniqueCount {
		panic(ErrTooMany("uniques, maximum is %d", MaxTypeUniqueCount))
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

func (ub *uniquesBuilder) AddUnique(name QName, fields []FieldName, comment ...string) IUniquesBuilder {
	ub.addUnique(name, fields, comment...)
	return ub
}

func (ub *uniquesBuilder) SetUniqueField(name FieldName) IUniquesBuilder {
	ub.setUniqueField(name)
	return ub
}
