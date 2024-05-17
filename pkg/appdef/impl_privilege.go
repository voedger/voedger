/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// # Implements:
//   - IPrivilege
type privilege struct {
	comment
	kinds   set.Set[PrivilegeKind]
	granted bool
	on      QNames
	fields  []FieldName
	role    *role
}

func newPrivilege(kind []PrivilegeKind, granted bool, on []QName, fields []FieldName, role *role, comment ...string) *privilege {
	pk := set.From(kind...)
	if pk.Len() == 0 {
		panic(ErrMissed("privilege kinds"))
	}

	names, err := validatePrivilegeOnNames(role.app, on...)
	if err != nil {
		panic(err)
	}

	o := role.app.Type(names[0])
	allPk := allPrivilegesOnType(o)
	if !allPk.ContainsAll(pk.AsArray()...) {
		panic(ErrIncompatible("privilege «%s» with %v", pk, o))
	}

	if len(fields) > 0 {
		if !pk.ContainsAny(PrivilegeKind_Select, PrivilegeKind_Update) {
			panic(ErrIncompatible("fields are not applicable for privilege «%s»", pk))
		}
		if err := validatePrivilegeOnFieldNames(role.app, on, fields); err != nil {
			panic(err)
		}
	}

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
	names, err := validatePrivilegeOnNames(role.app, on...)
	if err != nil {
		panic(err)
	}

	pk := allPrivilegesOnType(role.app.Type(names[0]))

	return newPrivilege(pk.AsArray(), granted, names, nil, role, comment...)
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

func (g privilege) Kinds() []PrivilegeKind { return g.kinds.AsArray() }

func (g privilege) On() QNames { return g.on }

func (g privilege) To() IRole { return g.role }

func (g privilege) String() string {
	if g.granted {
		return fmt.Sprintf("grant %v on %v to %v", g.kinds, g.on, g.role)
	}
	return fmt.Sprintf("revoke %v on %v from %v", g.kinds, g.on, g.role)
}
