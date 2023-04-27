/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

type Schema struct {
	Name       appdef.QName
	Kind       appdef.SchemaKind
	Fields     []*Field     `json:",omitempty"`
	Containers []*Container `json:",omitempty"`
}

type Field struct {
	Name       string
	Kind       appdef.DataKind
	Required   bool `json:",omitempty"`
	Verifiable bool `json:",omitempty"`
}

type Container struct {
	Name      string
	Type      appdef.QName
	MinOccurs appdef.Occurs
	MaxOccurs appdef.Occurs
}
