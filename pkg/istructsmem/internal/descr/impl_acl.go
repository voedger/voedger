/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newACL() *ACL {
	return &ACL{}
}

func (acl *ACL) read(a appdef.IWithACL, withPrincipals bool) {
	for r := range a.ACL {
		ar := newACLRule()
		ar.read(r, withPrincipals)
		*acl = append(*acl, ar)
	}
}

func newACLRule() *ACLRule {
	return &ACLRule{
		Ops:       make([]string, 0),
		Resources: newACLResourcePattern(),
	}
}

func (ar *ACLRule) read(acl appdef.IACLRule, withPrincipal bool) {
	ar.Comment = readComment(acl)
	ar.Policy = acl.Policy().TrimString()
	for _, k := range acl.Ops() {
		ar.Ops = append(ar.Ops, k.TrimString())
	}
	ar.Resources.read(acl.Resources())

	if withPrincipal {
		n := acl.Principal().QName()
		ar.Principal = &n
	}
}

func newACLResourcePattern() *ACLResourcePattern {
	return &ACLResourcePattern{
		Fields: make([]appdef.FieldName, 0),
	}
}

func (arp *ACLResourcePattern) read(rp appdef.IResourcePattern) {
	arp.On.Add(rp.On()...)
	arp.Fields = append(arp.Fields, rp.Fields()...)
}
