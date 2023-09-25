/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newType() *Type {
	return &Type{
		Fields:     make([]*Field, 0),
		Containers: make([]*Container, 0),
		Uniques:    make([]*Unique, 0),
	}
}

func (t *Type) read(typ appdef.IType) {
	t.Comment = readComment(typ)

	t.Name = typ.QName()
	t.Kind = typ.Kind()

	if fld, ok := typ.(appdef.IFields); ok {
		fld.Fields(func(field appdef.IField) {
			f := newField()
			f.read(field)
			t.Fields = append(t.Fields, f)
		})
	}

	if cnt, ok := typ.(appdef.IContainers); ok {
		cnt.Containers(func(cont appdef.IContainer) {
			c := newContainer()
			c.read(cont)
			t.Containers = append(t.Containers, c)
		})
	}

	if uni, ok := typ.(appdef.IUniques); ok {
		uni.Uniques(func(unique appdef.IUnique) {
			u := newUnique()
			u.read(unique)
			t.Uniques = append(t.Uniques, u)
		})
		if uf := uni.UniqueField(); uf != nil {
			t.UniqueField = uf.Name()
		}
	}

	if cDoc, ok := typ.(appdef.ICDoc); ok {
		if cDoc.Singleton() {
			t.Singleton = true
		}
	}
}

func newField() *Field { return &Field{} }

func (f *Field) read(field appdef.IField) {
	f.Comment = readComment(field)

	f.Name = field.Name()
	f.Kind = field.DataKind()
	f.Required = field.Required()
	f.Verifiable = field.Verifiable()
	if ref, ok := field.(appdef.IRefField); ok {
		for _, r := range ref.Refs() {
			f.Refs = append(f.Refs, r.String())
		}
	}

	switch field.DataKind() {
	case appdef.DataKind_string, appdef.DataKind_bytes:
		if sf, ok := field.(appdef.IStringField); ok {
			r := FieldRestricts{}
			if m := sf.Restricts().MinLen(); m != 0 {
				r.MinLen = m
				f.Restricts = &r
			}
			if m := sf.Restricts().MaxLen(); m != appdef.DefaultFieldMaxLength {
				r.MaxLen = m
				f.Restricts = &r
			}
			if p := sf.Restricts().Pattern(); p != nil {
				r.Pattern = p.String()
				f.Restricts = &r
			}
		}
	}
}

func newContainer() *Container { return &Container{} }

func (c *Container) read(cont appdef.IContainer) {
	c.Comment = readComment(cont)

	c.Name = cont.Name()
	c.Type = cont.QName()
	c.MinOccurs = cont.MinOccurs()
	c.MaxOccurs = cont.MaxOccurs()
}

func newUnique() *Unique { return &Unique{} }

func (u *Unique) read(unique appdef.IUnique) {
	u.Comment = readComment(unique)

	u.Name = unique.Name()
	for _, f := range unique.Fields() {
		u.Fields = append(u.Fields, f.Name())
	}
}
