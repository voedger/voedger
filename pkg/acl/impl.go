/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package acl

// acl struct will implement the IACL interface.
type acl[Role, Operation, Resource comparable] struct {
	permissions map[Role]map[Operation]map[Resource]bool
}

// HasPermission checks if the specified combination was granted via IACLBuilder.Grant() call.
func (a *acl[Role, Operation, Resource]) HasPermission(role Role, op Operation, res Resource) bool {
	if ops, ok := a.permissions[role]; ok {
		if resources, ok := ops[op]; ok {
			return resources[res]
		}
	}
	return false
}

// aclBuilder struct will implement the IACLBuilder interface.
type aclBuilder[Role, Operation, Resource comparable] struct {
	permissions map[Role]map[Operation]map[Resource]bool
}

// Grant adds a permission to the aclBuilder.
func (b *aclBuilder[Role, Operation, Resource]) Grant(role Role, op Operation, res Resource) {
	if _, ok := b.permissions[role]; !ok {
		b.permissions[role] = make(map[Operation]map[Resource]bool)
	}

	if _, ok := b.permissions[role][op]; !ok {
		b.permissions[role][op] = make(map[Resource]bool)
	}

	b.permissions[role][op][res] = true
}

// Build creates an acl instance with the current state of permissions.
func (b *aclBuilder[Role, Operation, Resource]) Build() IACL[Role, Operation, Resource] {
	return &acl[Role, Operation, Resource]{permissions: b.permissions}
}
