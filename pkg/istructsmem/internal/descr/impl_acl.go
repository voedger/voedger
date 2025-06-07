/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import (
	"github.com/voedger/voedger/pkg/appdef"
)

func newACL() *ACL {
	return &ACL{}
}

func (acl *ACL) read(a appdef.IWithACL, withPrincipals bool) {
	for _, r := range a.ACL() {
		ar := newACLRule()
		ar.read(r, withPrincipals)
		*acl = append(*acl, ar)
	}
}

func newACLRule() *ACLRule {
	return &ACLRule{
		Ops: make([]string, 0),
	}
}

func (ar *ACLRule) read(acl appdef.IACLRule, withPrincipal bool) {
	ar.Comment = readComment(acl)
	ar.Policy = acl.Policy().TrimString()
	for _, k := range acl.Ops() {
		ar.Ops = append(ar.Ops, k.TrimString())
	}
	ar.Filter.read(acl.Filter())

	if withPrincipal {
		n := acl.Principal().QName()
		ar.Principal = &n
	}
}

func (f *ACLFilter) read(flt appdef.IACLFilter) {
	f.Filter.read(flt)
	f.Fields = flt.Fields()
}
