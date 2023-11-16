/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

type IACL[Resource, Operation any, Role comparable] interface {

	// HasPermission checks if the specified operation, resource, and role combination
	// matches any of the permissions granted via IACLBuilder.Grant() calls.
	// Implementations should ideally index by Role to enhance performance.
	HasPermission(o Operation, r Resource, role Role) bool
}

type IACLBuilder[Resource, ResourcePattern, Operation, OperationPattern any, Role comparable] interface {
	Grant(op OperationPattern, rp ResourcePattern, role Role)
	Build(rm IResourceMatcher[Resource, ResourcePattern], om IOperationMatcher[Operation, OperationPattern]) IACL[Resource, Operation, Role]
}

type IResourceMatcher[Resource, ResourcePattern any] interface {
	Match(r Resource, rp ResourcePattern) bool
}

type IOperationMatcher[Operation, OperationPattern any] interface {
	Match(o Operation, op OperationPattern) bool
}
