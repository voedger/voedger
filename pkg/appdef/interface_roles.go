/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IRole interface {
	IType
}

type IRoleBuilder interface {
	ITypeBuilder
}

type IWithRoles interface {
	// Returns Role by name.
	//
	// Returns nil if not found.
	Role(name QName) IRole

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
