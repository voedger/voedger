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
	}
}

func (d *Def) readAppDef(def appdef.IDef) {
	d.Name = def.QName()
	d.Kind = def.Kind()

	def.Fields(func(field appdef.IField) {
		f := newField()
		f.Name = field.Name()
		f.Kind = field.DataKind()
		f.Required = field.Required()
		f.Verifiable = field.Verifiable()
		d.Fields = append(d.Fields, f)
	})

	def.Containers(func(cont appdef.IContainer) {
		c := newContainer()
		c.Name = cont.Name()
		c.Type = cont.Def()
		c.MinOccurs = cont.MinOccurs()
		c.MaxOccurs = cont.MaxOccurs()
		d.Containers = append(d.Containers, c)
	})
}

func newField() *Field { return &Field{} }

func newContainer() *Container { return &Container{} }
