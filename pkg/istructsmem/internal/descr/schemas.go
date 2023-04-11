/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

type Schema struct {
	Name       istructs.QName
	Kind       istructs.SchemaKindType
	Fields     []*Field     `json:",omitempty"`
	Containers []*Container `json:",omitempty"`
}

type Field struct {
	Name       string
	Kind       istructs.DataKindType
	Required   bool `json:",omitempty"`
	Verifiable bool `json:",omitempty"`
}

type Container struct {
	Name      string
	Type      istructs.QName
	MinOccurs istructs.ContainerOccursType
	MaxOccurs istructs.ContainerOccursType
}
