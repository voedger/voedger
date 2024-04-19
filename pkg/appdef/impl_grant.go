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
//   - IGRant
type grant struct {
	comment
	kind    GrantKind
	objects QNames
	fields  []FieldName
	role    *role
}

func newGrant(kind GrantKind, objects []QName, fields []FieldName, role *role, comment ...string) *grant {
	g := &grant{
		comment: makeComment(comment...),
		kind:    kind,
		objects: QNamesFrom(objects...),
		fields:  fields, // TODO: check fields validity
		role:    role,
	}
	return g
}

func (g grant) Fields() []FieldName { return g.fields }

func (g grant) Kind() GrantKind { return g.kind }

func (g grant) Objects() QNames { return g.objects }

func (g grant) Role() IRole { return g.role }

func (g grant) String() string {
	return fmt.Sprintf("grant %s to %v for %v", g.kind.TrimString(), g.objects, g.role)
}

func (k GrantKind) TrimString() string {
	const pref = "GrantKind_"
	return strings.TrimPrefix(k.String(), pref)
}
