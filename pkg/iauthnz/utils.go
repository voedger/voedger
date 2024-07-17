/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"slices"
)

func IsSystemRole(role appdef.QName) bool {
	return slices.Contains(SysRoles, role)
}

// returns NullQName if missing
func QNameAncestor(qName appdef.QName) appdef.QName {
	return rolesInheritance[qName]
}

func IsSystemPrincipal(principals []Principal, wsid istructs.WSID) bool {
	for _, p := range principals {
		if p.Kind == PrincipalKind_Role && p.WSID == wsid && p.QName == QNameRoleSystem {
			return true
		}
	}
	return false
}
