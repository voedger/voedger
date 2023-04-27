/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newSchema() *Schema {
	return &Schema{
		Fields:     make([]*Field, 0),
		Containers: make([]*Container, 0),
	}
}

func (s *Schema) readAppSchema(schema appdef.Schema) {
	s.Name = schema.QName()
	s.Kind = schema.Kind()

	schema.Fields(func(field appdef.Field) {
		f := newField()
		f.Name = field.Name()
		f.Kind = field.DataKind()
		f.Required = field.Required()
		f.Verifiable = field.Verifiable()
		s.Fields = append(s.Fields, f)
	})

	schema.Containers(func(cont appdef.Container) {
		c := newContainer()
		c.Name = cont.Name()
		c.Type = cont.Schema()
		c.MinOccurs = cont.MinOccurs()
		c.MaxOccurs = cont.MaxOccurs()
		s.Containers = append(s.Containers, c)
	})
}

func newField() *Field { return &Field{} }

func newContainer() *Container { return &Container{} }
