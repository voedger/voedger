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

	def.Fields(func(field appdef.IField) {
		f := newField()
		f.read(field)
		d.Fields = append(d.Fields, f)
	})

	def.Containers(func(cont appdef.IContainer) {
		c := newContainer()
		c.read(cont)
		d.Containers = append(d.Containers, c)
	})

	def.Uniques(func(name string, fields []appdef.IField) {
		u := newUnique()
		u.read(name, fields)
		d.Uniques = append(d.Uniques, u)
	})
}

func newField() *Field { return &Field{} }

func (f *Field) read(field appdef.IField) {
	f.Name = field.Name()
	f.Kind = field.DataKind()
	f.Required = field.Required()
	f.Verifiable = field.Verifiable()
}

func newContainer() *Container { return &Container{} }

func (c *Container) read(cont appdef.IContainer) {
	c.Name = cont.Name()
	c.Type = cont.Def()
	c.MinOccurs = cont.MinOccurs()
	c.MaxOccurs = cont.MaxOccurs()
}

func newUnique() *Unique { return &Unique{} }

func (u *Unique) read(name string, fields []appdef.IField) {
	u.Name = name
	for _, f := range fields {
		u.Fields = append(u.Fields, f.Name())
	}
}
