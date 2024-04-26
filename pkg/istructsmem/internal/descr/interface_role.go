/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Role struct {
	Type
	Privileges []*Privilege
}

type Privilege struct {
	Comment string `json:",omitempty"`
	Access  string // `grant` or `revoke`
	Kinds   []string
	On      appdef.QNames
	Fields  []string `json:",omitempty"`
}
