/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package uniques

import (
	"fmt"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
	"github.com/voedger/voedger/pkg/appdef/internal/slicex"
)

// # Supports:
//   - appdef.IUnique
type Unique struct {
	comments.WithComments
	name   appdef.QName
	fields []appdef.IField
}

func NewUnique(name appdef.QName, fieldNames []appdef.FieldName, fields appdef.IWithFields) *Unique {
	u := &Unique{
		name:   name,
		fields: make([]appdef.IField, 0),
	}
	slices.Sort(fieldNames)
	for _, f := range fieldNames {
		fld := fields.Field(f)
		if fld == nil {
			panic(appdef.ErrFieldNotFound(f))
		}
		u.fields = append(u.fields, fld)
	}
	return u
}

func (u Unique) Name() appdef.QName {
	return u.name
}

func (u Unique) Fields() []appdef.IField {
	return u.fields
}

func (u Unique) String() string {
	return fmt.Sprintf("unique «%v»", u.name)
}

// # Supports:
//   - appdef.IWithUniques
type WithUniques struct {
	find    appdef.FindType
	fields  appdef.IWithFields
	uniques map[appdef.QName]appdef.IUnique
	field   appdef.IField
}

func MakeWithUniques(find appdef.FindType, fields appdef.IWithFields) WithUniques {
	uu := WithUniques{
		find:    find,
		fields:  fields,
		uniques: make(map[appdef.QName]appdef.IUnique),
	}
	return uu
}

func (uu *WithUniques) setUniqueField(name appdef.FieldName) {
	if name == appdef.NullName {
		uu.field = nil
		return
	}
	if ok, err := appdef.ValidFieldName(name); !ok {
		panic(fmt.Errorf("unique field name «%v» is invalid: %w", name, err))
	}

	fld := uu.fields.Field(name)
	if fld == nil {
		panic(appdef.ErrFieldNotFound(name))
	}

	uu.field = fld
}

func (uu WithUniques) UniqueByName(name appdef.QName) appdef.IUnique {
	if u, ok := uu.uniques[name]; ok {
		return u
	}
	return nil
}

func (uu WithUniques) UniqueCount() int {
	return len(uu.uniques)
}

func (uu WithUniques) UniqueField() appdef.IField {
	return uu.field
}

func (uu WithUniques) Uniques() map[appdef.QName]appdef.IUnique {
	return uu.uniques
}

func (uu *WithUniques) addUnique(name appdef.QName, fields []appdef.FieldName, comment ...string) {
	if name == appdef.NullQName {
		panic(appdef.ErrMissed("unique name"))
	}
	if ok, err := appdef.ValidQName(name); !ok {
		panic(fmt.Errorf("unique name «%v» is invalid: %w", name, err))
	}
	if uu.UniqueByName(name) != nil {
		panic(appdef.ErrAlreadyExists("unique «%v»", name))
	}

	if t := uu.find(name); t.Kind() != appdef.TypeKind_null {
		panic(appdef.ErrAlreadyExists("name «%v» already used for %v", name, t))
	}

	if len(fields) == 0 {
		panic(appdef.ErrMissed("unique «%v» fields", name))
	}
	if i, j := slicex.FindDuplicates(fields); i >= 0 {
		panic(appdef.ErrAlreadyExists("fields in unique «%v» has duplicates (fields[%d] == fields[%d] == %q)", name, i, j, fields[i]))
	}

	if len(fields) > appdef.MaxTypeUniqueFieldsCount {
		panic(appdef.ErrTooMany("fields in unique «%v», maximum is %d", name, appdef.MaxTypeUniqueFieldsCount))
	}

	for _, un := range uu.uniques {
		ff := make([]appdef.FieldName, 0)
		for _, f := range un.Fields() {
			ff = append(ff, f.Name())
		}
		if slicex.Overlaps(fields, ff) {
			panic(appdef.ErrAlreadyExists("type already has %v which fields overlaps new unique fields", un))
		}
	}

	if len(uu.uniques) >= appdef.MaxTypeUniqueCount {
		panic(appdef.ErrTooMany("uniques, maximum is %d", appdef.MaxTypeUniqueCount))
	}

	un := NewUnique(name, fields, uu.fields)

	comments.SetComment(&un.WithComments, comment...)

	uu.uniques[name] = un
}

// # Supports:
//   - appdef.IUniquesBuilder
type UniquesBuilder struct {
	*WithUniques
}

func MakeUniquesBuilder(uniques *WithUniques) UniquesBuilder {
	return UniquesBuilder{WithUniques: uniques}
}

func (ub *UniquesBuilder) AddUnique(name appdef.QName, fields []appdef.FieldName, comment ...string) appdef.IUniquesBuilder {
	ub.addUnique(name, fields, comment...)
	return ub
}

func (ub *UniquesBuilder) SetUniqueField(name appdef.FieldName) appdef.IUniquesBuilder {
	ub.setUniqueField(name)
	return ub
}
