/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IRole interface {
	IType

	IWithPrivileges
}

type IRoleBuilder interface {
	ITypeBuilder

	// Adds new privilege with specified kind on specified objects.
	//
	// # Panics:
	//   - if kind is PrivilegeKind_null,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if fields contains unknown names.
	Grant(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder

	// Grants all available privileges on specified objects.
	//
	// If the objects are records or view records, then insert, update, and select privileges are granted.
	//
	// If the objects are commands or queries, their execution is granted.
	//
	// If the objects are workspaces, then:
	//	- insert, update and select records and view records of these workspaces are granted,
	//	- execution of commands & queries from these workspaces is granted.
	//
	// If the objects are roles, then all privileges from these roles are granted.
	GrantAll(on []QName, comment ...string) IRoleBuilder

	// Revokes privilege with specified kind on specified objects.
	//
	// # Panics:
	//   - if kind is PrivilegeKind_null,
	//	 - if objects are empty,
	//	 - if objects contains unknown names,
	//	 - if fields contains unknown names,
	//   - if role is unknown.
	Revoke(kind PrivilegeKind, on []QName, fields []FieldName, comment ...string) IRoleBuilder
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
