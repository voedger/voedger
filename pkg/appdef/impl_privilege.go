/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strings"
)

// # Implements:
//   - IPrivilege
type privilege struct {
	comment
	kind    PrivilegeKind
	granted bool
	on      QNames
	fields  []FieldName
	role    *role
}

func newPrivilege(kind PrivilegeKind, granted bool, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	g := &privilege{
		comment: makeComment(comment...),
		granted: granted,
		kind:    kind,
		on:      QNamesFrom(on...),
		fields:  fields, // TODO: check fields validity
		role:    role,
	}
	return g
}

func newGrant(kind PrivilegeKind, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	return newPrivilege(kind, true, on, fields, role, comment...)
}

func newRevoke(kind PrivilegeKind, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	return newPrivilege(kind, false, on, fields, role, comment...)
}

func (g privilege) Fields() []FieldName { return g.fields }

func (g privilege) IsGranted() bool { return g.granted }

func (g privilege) IsRevoked() bool { return !g.granted }

func (g privilege) Kind() PrivilegeKind { return g.kind }

func (g privilege) On() QNames { return g.on }

func (g privilege) To() IRole { return g.role }

func (g privilege) String() string {
	if g.granted {
		return fmt.Sprintf("grant %s on %v to %v", g.kind.TrimString(), g.on, g.role)
	}
	return fmt.Sprintf("revoke %s on %v from %v", g.kind.TrimString(), g.on, g.role)
}

func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}
