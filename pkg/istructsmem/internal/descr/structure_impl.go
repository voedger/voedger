/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newStructure() *Structure {
	return &Structure{
		Fields:     make([]*Field, 0),
		Containers: make([]*Container, 0),
		Uniques:    make([]*Unique, 0),
	}
}

func (t *Structure) read(str appdef.IStructure) {
	t.Comment = readComment(str)

	t.QName = str.QName()
	t.Kind = str.Kind()

	str.Fields(func(field appdef.IField) {
		f := newField()
		f.read(field)
		t.Fields = append(t.Fields, f)
	})

	str.Containers(func(cont appdef.IContainer) {
		c := newContainer()
		c.read(cont)
		t.Containers = append(t.Containers, c)
	})

	str.Uniques(func(unique appdef.IUnique) {
		u := newUnique()
		u.read(unique)
		t.Uniques = append(t.Uniques, u)
	})
	if uf := str.UniqueField(); uf != nil {
		t.UniqueField = uf.Name()
	}

	if cDoc, ok := str.(appdef.ICDoc); ok {
		if cDoc.Singleton() {
			t.Singleton = true
		}
	}
}

func newField() *Field { return &Field{} }

func (f *Field) read(field appdef.IField) {
	f.Comment = readComment(field)

	f.Name = field.Name()
	if q := field.Data().QName(); q != appdef.NullQName {
		f.Data = &q
	} else {
		f.DataType = newData()
		f.DataType.read(field.Data())
	}
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
