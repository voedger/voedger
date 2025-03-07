/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// Returns recursive list of role ancestors for specified role in the specified workspace.
//
// Role inheritance provided by `GRANT <role> TO <role>` statement.
// These inheritances statements can be provided in the specified workspace or in any of its ancestors.
//
// If role has no ancestors, then result contains only specified role.
// Result is alphabetically sorted list of role names.
func RecursiveRoleAncestors(role appdef.IRole, ws appdef.IWorkspace) (roles appdef.QNames) {
	roles.Add(role.QName())

	for _, acl := range ws.ACL() {
		if acl.Op(appdef.OperationKind_Inherits) && (acl.Principal() == role) {
			for _, t := range appdef.FilterMatches(acl.Filter(), ws.Types()) {
				if r, ok := t.(appdef.IRole); ok {
					roles.Add(RecursiveRoleAncestors(r, ws)...)
				}
			}
		}
	}

	for _, w := range ws.Ancestors() {
		roles.Add(RecursiveRoleAncestors(role, w)...)
	}

	return roles
}
