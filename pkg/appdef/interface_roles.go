/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IRole interface {
	IType

	IWithGrants
}

type IRoleBuilder interface {
	ITypeBuilder

	// Adds new Grant with specified kind to specified objects.
	//
	// # Panics:
	//   - if kind is GrantKind_null,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if fields contains unknown names.
	Grant(kind GrantKind, objects []QName, fields []FieldName, comment ...string) IRoleBuilder

	// Adds all available grants to specified objects.
	//
	// If the objects are tables, then insert, update, and select operations are granted.
	//
	// If the objects are commands or queries, their execution is allowed.
	//
	// If the objects are workspaces, then:
	//	- insert, update and select from the tables of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	GrantAll(objects []QName, comment ...string) IRoleBuilder

	// Adds new Grant with GrantKind_Role to specified roles.
	GrantRoles(roles []QName, comment ...string) IRoleBuilder
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
