/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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
	role.appendGrant(g)
	return g
}

func (g grant) Fields() []FieldName { return g.fields }

func (g grant) Kind() GrantKind { return g.kind }

func (g grant) Objects() QNames { return g.objects }

func (g grant) Role() IRole { return g.role }
