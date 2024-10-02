/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type ACL []*ACLRule

type ACLRule struct {
	Comment   string `json:",omitempty"`
	Policy    string // `Allow` or `Deny`
	Ops       []string
	Resources *ACLResourcePattern
	Principal *appdef.QName `json:",omitempty"`
}

type ACLResourcePattern struct {
	On     appdef.QNames
	Fields []appdef.FieldName `json:",omitempty"`
}
