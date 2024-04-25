/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

func newRole() *Role {
	return &Role{
		Privileges: make([]*Privilege, 0),
	}
}

func (r *Role) read(role appdef.IRole) {
	r.Type.read(role)
	role.Privileges(func(priv appdef.IPrivilege) {
		p := newPrivilege()
		p.read(priv)
		r.Privileges = append(r.Privileges, p)
	})
}

func newPrivilege() *Privilege {
	return &Privilege{}
}

func (p *Privilege) read(priv appdef.IPrivilege) {
	p.Comment = readComment(priv)
	p.Access = appdef.PrivilegeAccessControlString(priv.IsGranted())
	for _, k := range priv.Kinds() {
		p.Kinds = append(p.Kinds, k.TrimString())
	}
	p.On = priv.On()
	p.Fields = priv.Fields()
}
