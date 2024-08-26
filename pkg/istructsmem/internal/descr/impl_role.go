/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newRole() *Role {
	return &Role{
		ACL: make([]*ACLRule, 0),
	}
}

func (r *Role) read(role appdef.IRole) {
	r.Type.read(role)
	role.ACL(func(acl appdef.IACLRule) {
		ar := newACLRule()
		ar.read(acl)
		r.ACL = append(r.ACL, ar)
	})
}

func newACLRule() *ACLRule {
	return &ACLRule{
		Ops:       make([]string, 0),
		Resources: newACLResourcePattern(),
	}
}

func (ar *ACLRule) read(acl appdef.IACLRule) {
	ar.Comment = readComment(acl)
	ar.Policy = acl.Policy().TrimString()
	for _, k := range acl.Ops() {
		ar.Ops = append(ar.Ops, k.TrimString())
	}
	ar.Resources.read(acl.Resources())
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
