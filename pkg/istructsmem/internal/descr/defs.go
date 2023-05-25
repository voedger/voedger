/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

type Def struct {
	Name        appdef.QName
	Kind        appdef.DefKind
	Fields      []*Field     `json:",omitempty"`
	Containers  []*Container `json:",omitempty"`
	Uniques     []*Unique    `json:",omitempty"`
	UniqueField string       `json:",omitempty"`
	Singleton   bool         `json:",omitempty"`
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

type Unique struct {
	Name   string
	Fields []string
}
