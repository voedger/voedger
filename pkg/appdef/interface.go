/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Application definition is a set of types, views, commands, queries and workspaces.
type IAppDef interface {
	IWithComments

	IWithPackages
	IWithWorkspaces
	IWithACL

	// Returns type by name.
	//
	// If not found then empty type with TypeKind_null is returned
	Type(QName) IType

	// Enumerates types.
	//
	// Types are enumerated in alphabetical order of QNames.
	Types(func(IType) bool)
}

type IAppDefBuilder interface {
	ICommentsBuilder

	IPackagesBuilder
	IWorkspacesBuilder

	// Returns application definition while building.
	//
	// Can be called before or after all entities added.
	// Does not validate application definition, some types may be invalid.
	AppDef() IAppDef

	// Builds application definition.
	//
	// Validates and returns builded application type or error.
	// Must be called after all entities added.
	Build() (IAppDef, error)

	// Builds application definition.
	//
	// Validates and returns builded application type.
	// Must be called after all entities added.
	//
	// # Panics if error occurred.
	MustBuild() IAppDef
}
