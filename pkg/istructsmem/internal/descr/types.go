/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

type Type struct {
	Comment     string `json:",omitempty"`
	Name        appdef.QName
	Kind        appdef.TypeKind
	Fields      []*Field     `json:",omitempty"`
	Containers  []*Container `json:",omitempty"`
	Uniques     []*Unique    `json:",omitempty"`
	UniqueField string       `json:",omitempty"`
	Singleton   bool         `json:",omitempty"`
}

type Field struct {
	Comment    string `json:",omitempty"`
	Name       string
	Kind       appdef.DataKind
	Required   bool            `json:",omitempty"`
	Verifiable bool            `json:",omitempty"`
	Refs       []string        `json:",omitempty"`
	Restricts  *FieldRestricts `json:",omitempty"`
}

type FieldRestricts struct {
	MinLen  uint16 `json:",omitempty"`
	MaxLen  uint16 `json:",omitempty"`
	Pattern string `json:",omitempty"`
}

type Container struct {
	Comment   string `json:",omitempty"`
	Name      string
	Type      appdef.QName
	MinOccurs appdef.Occurs
	MaxOccurs appdef.Occurs
}

type Unique struct {
	Comment string `json:",omitempty"`
	Name    string
	Fields  []string
}
