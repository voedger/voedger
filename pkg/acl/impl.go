/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

// acl is the implementation of the IACL interface.
type acl[Resource, Operation, ResourcePattern, OperationPattern any, Role comparable] struct {
	permissions map[Role][]patternedPermission[ResourcePattern, OperationPattern]
	rm          IResourceMatcher[Resource, ResourcePattern]
	om          IOperationMatcher[Operation, OperationPattern]
}

// patternedPermission represents a combination of an operation pattern, a resource pattern.
type patternedPermission[ResourcePattern, OperationPattern any] struct {
	operationPattern OperationPattern
	resourcePattern  ResourcePattern
}

// NewACL creates a new instance of acl.
func NewACL[Resource, Operation, ResourcePattern, OperationPattern any, Role comparable](rm IResourceMatcher[Resource, ResourcePattern], om IOperationMatcher[Operation, OperationPattern]) *acl[Resource, Operation, ResourcePattern, OperationPattern, Role] {
	return &acl[Resource, Operation, ResourcePattern, OperationPattern, Role]{
		permissions: make(map[Role][]patternedPermission[ResourcePattern, OperationPattern]),
		rm:          rm,
		om:          om,
	}
}

// HasPermission checks if the specified operation, resource, and role combination
// matches any of the permissions granted.
func (a *acl[Resource, Operation, ResourcePattern, OperationPattern, Role]) HasPermission(o Operation, r Resource, role Role) bool {
	perms, ok := a.permissions[role]
	if !ok {
		return false
	}

	for _, perm := range perms {
		if a.rm.Match(r, perm.resourcePattern) && a.om.Match(o, perm.operationPattern) {
			return true
		}
	}

	return false
}

// aclBuilder is the implementation of the IACLBuilder interface.
type aclBuilder[Resource, ResourcePattern, Operation, OperationPattern any, Role comparable] struct {
	grants []grant[OperationPattern, ResourcePattern, Role]
}

// grant represents a combination of an operation pattern, a resource pattern, and a role.
type grant[OperationPattern, ResourcePattern any, Role comparable] struct {
	operationPattern OperationPattern
	resourcePattern  ResourcePattern
	role             Role
}

// NewACLBuilder creates a new instance of aclBuilder.
func NewACLBuilder[Resource, ResourcePattern, Operation, OperationPattern any, Role comparable]() *aclBuilder[Resource, ResourcePattern, Operation, OperationPattern, Role] {
	return &aclBuilder[Resource, ResourcePattern, Operation, OperationPattern, Role]{}
}

// Grant adds a new grant to the aclBuilder.
func (b *aclBuilder[Resource, ResourcePattern, Operation, OperationPattern, Role]) Grant(op OperationPattern, rp ResourcePattern, role Role) {
	b.grants = append(b.grants, grant[OperationPattern, ResourcePattern, Role]{op, rp, role})
}

// Build constructs an acl instance from the grants and matchers.
func (b *aclBuilder[Resource, ResourcePattern, Operation, OperationPattern, Role]) Build(rm IResourceMatcher[Resource, ResourcePattern], om IOperationMatcher[Operation, OperationPattern]) IACL[Resource, Operation, Role] {
	newACL := NewACL[Resource, Operation, ResourcePattern, OperationPattern, Role](rm, om)
	for _, g := range b.grants {
		newACL.permissions[g.role] = append(newACL.permissions[g.role], patternedPermission[ResourcePattern, OperationPattern]{g.operationPattern, g.resourcePattern})
	}
	return newACL
}
