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
	objects QNames
	fields  []FieldName
	role    *role
}

func newGrant(kind PrivilegeKind, objects []QName, fields []FieldName, role *role, comment ...string) *privilege {
	g := &privilege{
		comment: makeComment(comment...),
		kind:    kind,
		objects: QNamesFrom(objects...),
		fields:  fields, // TODO: check fields validity
		role:    role,
	}
	return g
}

func (g privilege) Fields() []FieldName { return g.fields }

func (g privilege) Kind() PrivilegeKind { return g.kind }

func (g privilege) Objects() QNames { return g.objects }

func (g privilege) Role() IRole { return g.role }

func (g privilege) String() string {
	return fmt.Sprintf("grant %s to %v for %v", g.kind.TrimString(), g.objects, g.role)
}

func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}
