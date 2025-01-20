/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// # Supports:
//   - appdef.IWithACL
type WithACL struct {
	acl []appdef.IACLRule
}

func MakeWithACL() WithACL {
	return WithACL{acl: make([]appdef.IACLRule, 0)}
}

func (acl WithACL) ACL() []appdef.IACLRule { return acl.acl }

func (acl *WithACL) AppendACL(r appdef.IACLRule) { acl.acl = append(acl.acl, r) }
