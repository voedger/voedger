/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "iter"

type IRole interface {
	IType

	IWithACL

	// Returns all roles that this role inherits.
	//
	// Role inheritance provided by `GRANT <role> TO <role>` statement.
	//
	// Only direct inheritance is returned. If role inherits another role, which inherits another role, then only direct ancestor is returned.
	Ancestors() iter.Seq[QName]
}

type IRoleBuilder interface {
	ITypeBuilder

	// Grants operations on filtered types.
	//
	// # Panics:
	//   - if ops is empty,
	//	 - if ops contains incompatible operations (e.g. INSERT with EXECUTE),
	//	 - if filtered type is not compatible with operations,
	//	 - if fields contains unknown names.
	Grant(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) IRoleBuilder

	// Grants all available operations on filtered types.
	//
	// If the types are records or view records, then insert, update, and select are granted.
	//
	// If the types are commands or queries, their execution is granted.
	//
	// If the types are roles, then inheritance all operations from these roles are granted.
	GrantAll(flt IFilter, comment ...string) IRoleBuilder

	// Revokes operations on filtered types.
	Revoke(ops []OperationKind, flt IFilter, fields []FieldName, comment ...string) IRoleBuilder

	// Remove all available operations on filtered types.
	RevokeAll(flt IFilter, comment ...string) IRoleBuilder
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
