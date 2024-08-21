/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IRole interface {
	IType

	IWithACL
}

type IRoleBuilder interface {
	ITypeBuilder

	// Grants specified operations on specified resources.
	//
	// # Panics:
	//   - if ops is empty,
	//	 - if resources are empty,
	//	 - if resources contains unknown names,
	//	 - if ops are not compatible with resources,
	//	 - if fields contains unknown names.
	Grant(ops []OperationKind, resources []QName, fields []FieldName, comment ...string) IRoleBuilder

	// Grants all available operations on specified resources.
	//
	// If the resources are records or view records, then insert, update, and select are granted.
	//
	// If the resources are commands or queries, their execution is granted.
	//
	// If the resources are workspaces, then:
	//	- insert, update and select records and view records of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	//
	// If the resources are roles, then all operations from these roles are granted.
	GrantAll(resources []QName, comment ...string) IRoleBuilder

	// Revokes operations on specified resources.
	Revoke(ops []OperationKind, resources []QName, comment ...string) IRoleBuilder

	// Remove all available operations on specified resources.
	RevokeAll(resources []QName, comment ...string) IRoleBuilder
}

type IWithRoles interface {
	// Returns Role by name.
	//
	// Returns nil if not found.
	Role(QName) IRole

	// Enumerates all roles
	//
	// Roles are enumerated in alphabetical order by QName
	Roles(func(IRole))
}

type IRolesBuilder interface {
	// Adds new Role type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddRole(QName) IRoleBuilder
}
