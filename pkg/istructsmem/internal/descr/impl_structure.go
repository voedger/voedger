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
		Uniques:    make(map[string]*Unique),
	}
}

func (s *Structure) read(str appdef.IStructure) {
	s.Type.read(str)

	s.Kind = str.Kind().TrimString()

	for _, fld := range str.Fields() {
		f := newField()
		f.read(fld)
		s.Fields = append(s.Fields, f)
	}

	for _, cont := range str.Containers() {
		c := newContainer()
		c.read(cont)
		s.Containers = append(s.Containers, c)
	}

	for n, unique := range str.Uniques() {
		u := newUnique()
		u.read(unique)
		s.Uniques[n.String()] = u
	}
	if uf := str.UniqueField(); uf != nil {
		s.UniqueField = uf.Name()
	}

	if singleton, ok := str.(appdef.ISingleton); ok {
		if singleton.Singleton() {
			s.Singleton = true
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
