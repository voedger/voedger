/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newDef() *Def {
	return &Def{
		Fields:     make([]*Field, 0),
		Containers: make([]*Container, 0),
		Uniques:    make([]*Unique, 0),
	}
}

func (d *Def) read(def appdef.IDef) {
	d.Name = def.QName()
	d.Kind = def.Kind()

	if fld, ok := def.(appdef.IFields); ok {
		fld.Fields(func(field appdef.IField) {
			f := newField()
			f.read(field)
			d.Fields = append(d.Fields, f)
		})
	}

	if cnt, ok := def.(appdef.IContainers); ok {
		cnt.Containers(func(cont appdef.IContainer) {
			c := newContainer()
			c.read(cont)
			d.Containers = append(d.Containers, c)
		})
	}

	if uni, ok := def.(appdef.IUniques); ok {
		uni.Uniques(func(unique appdef.IUnique) {
			u := newUnique()
			u.read(unique)
			d.Uniques = append(d.Uniques, u)
		})
		if uf := uni.UniqueField(); uf != nil {
			d.UniqueField = uf.Name()
		}
	}

	if cDoc, ok := def.(appdef.ICDoc); ok {
		if cDoc.Singleton() {
			d.Singleton = true
		}
	}
}

func newField() *Field { return &Field{} }

func (f *Field) read(field appdef.IField) {
	f.Name = field.Name()
	f.Kind = field.DataKind()
	f.Required = field.Required()
	f.Verifiable = field.Verifiable()
	if ref, ok := field.(appdef.IRefField); ok {
		for _, r := range ref.Refs() {
			f.Refs = append(f.Refs, r.String())
		}
	}
}

func newContainer() *Container { return &Container{} }

func (c *Container) read(cont appdef.IContainer) {
	c.Name = cont.Name()
	c.Type = cont.Def()
	c.MinOccurs = cont.MinOccurs()
	c.MaxOccurs = cont.MaxOccurs()
}

func newUnique() *Unique { return &Unique{} }

func (u *Unique) read(unique appdef.IUnique) {
	u.Name = unique.Name()
	for _, f := range unique.Fields() {
		u.Fields = append(u.Fields, f.Name())
	}
}
