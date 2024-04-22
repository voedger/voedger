/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
)

// # Implements:
//   - IPrivilege
type privilege struct {
	comment
	kinds   PrivilegeKinds
	granted bool
	on      QNames
	fields  []FieldName
	role    *role
}

func newPrivilege(kind []PrivilegeKind, granted bool, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	pk := PrivilegeKindsFrom(kind...)
	if len(pk) == 0 {
		panic(ErrPrivilegeKindsMissed)
	}

	names := QNamesFrom(on...)
	if len(names) == 0 {
		panic(ErrPrivilegeOnMissed)
	}

	// TODO: check pk compatibility with names

	g := &privilege{
		comment: makeComment(comment...),
		granted: granted,
		kinds:   pk,
		on:      names,
		fields:  fields, // TODO: check fields validity
		role:    role,
	}
	return g
}

func newGrant(kinds []PrivilegeKind, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	return newPrivilege(kinds, true, on, fields, role, comment...)
}

func newRevoke(kinds []PrivilegeKind, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	return newPrivilege(kinds, false, on, fields, role, comment...)
}

func newPrivilegeAll(granted bool, on []QName, role *role, comment ...string) *privilege {
	names := QNamesFrom(on...)
	if len(names) == 0 {
		panic(ErrPrivilegeOnMissed)
	}

	pk := PrivilegeKinds{}

	o := names[0]
	t := role.app.Type(o)
	if t == nil {
		panic(fmt.Errorf("%w: %v", ErrTypeNotFound, o))
	}

	switch t.Kind() {
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		pk = PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}
	case TypeKind_Command, TypeKind_Query:
		pk = PrivilegeKinds{PrivilegeKind_Execute}
	case TypeKind_Workspace:
		pk = PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute}
	case TypeKind_Role:
		pk = PrivilegeKinds{PrivilegeKind_Inherits}
	default:
		panic(fmt.Errorf("can not grant privileges on: %w: %v", ErrInvalidTypeKind, o))
	}

	return newPrivilege(pk, granted, names, nil, role, comment...)
}

func newGrantAll(on []QName, role *role, comment ...string) *privilege {
	return newPrivilegeAll(true, on, role, comment...)
}

func newRevokeAll(on []QName, role *role, comment ...string) *privilege {
	return newPrivilegeAll(false, on, role, comment...)
}

func (g privilege) Fields() []FieldName { return g.fields }

func (g privilege) IsGranted() bool { return g.granted }

func (g privilege) IsRevoked() bool { return !g.granted }

func (g privilege) Kinds() PrivilegeKinds { return g.kinds }

func (g privilege) On() QNames { return g.on }

func (g privilege) To() IRole { return g.role }

func (g privilege) String() string {
	if g.granted {
		return fmt.Sprintf("grant %v on %v to %v", g.kinds, g.on, g.role)
	}
	return fmt.Sprintf("revoke %v on %v from %v", g.kinds, g.on, g.role)
}
