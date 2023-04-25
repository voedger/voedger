/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/schemas"
)

type Schema struct {
	Name       schemas.QName
	Kind       schemas.SchemaKind
	Fields     []*Field     `json:",omitempty"`
	Containers []*Container `json:",omitempty"`
}

type Field struct {
	Name       string
	Kind       schemas.DataKind
	Required   bool `json:",omitempty"`
	Verifiable bool `json:",omitempty"`
}

type Container struct {
	Name      string
	Type      schemas.QName
	MinOccurs schemas.Occurs
	MaxOccurs schemas.Occurs
}
