/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IRole interface {
	IType

	// Unwanted type assertion stub
	IsRole()
}

type IRoleBuilder interface {
	ITypeBuilder
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
