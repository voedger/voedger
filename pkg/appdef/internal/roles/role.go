/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package roles

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IRole
type Role struct {
	types.Typ
}

func NewRole(ws appdef.IWorkspace, name appdef.QName) *Role {
	r := &Role{
		Typ: types.MakeType(ws.App(), ws, name, appdef.TypeKind_Role),
	}
	types.Propagate(r)
	return r
}

func (Role) IsRole() {}

// # Supports:
//   - appdef.IRoleBuilder
type RoleBuilder struct {
	types.TypeBuilder
	r *Role
}

func NewRoleBuilder(r *Role) *RoleBuilder {
	return &RoleBuilder{
		TypeBuilder: types.MakeTypeBuilder(&r.Typ),
		r:           r,
	}
}
