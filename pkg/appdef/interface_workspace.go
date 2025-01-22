/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workspace is a set of types.
type IWorkspace interface {
	IType
	IWithAbstract

	IWithACL

	// Returns ancestors workspaces.
	//
	// Ancestors are enumerated in alphabetic order.
	// Only direct ancestors are enumerated.
	// Workspace `sys.Workspace` is default ancestor used then no other ancestor is specified.
	Ancestors() []IWorkspace

	// Workspace descriptor document.
	// See [#466](https://github.com/voedger/voedger/issues/466)
	//
	// Descriptor is CDoc document.
	// If the Descriptor is an abstract document, the workspace must also be abstract.
	Descriptor() QName

	// Returns is workspace inherits from specified workspace.
	//
	// Returns true:
	// 	- if the workspace itself has the specified name or
	// 	- if one of the direct ancestors has the specified name or
	// 	- if one of the ancestors of the ancestors (recursively) has the specified name.
	Inherits(QName) bool

	// LocalType returns type by name. Find only in the workspace, not in ancestors or used workspaces.
	//
	// If not found then empty type with TypeKind_null is returned
	LocalType(QName) IType

	// LocalTypes returns all types defined in the workspace in alphabetical order.
	LocalTypes() []IType

	// Returns a type by name. All ancestor types are searched recursively.
	//
	// If the workspace uses other workspaces, these used workspaces (but not the types from them) can be found by this method.
	//
	// If not found then empty type with TypeKind_null is returned
	Type(QName) IType

	// Returns all types, include types from all ancestors recursively, in alphabetical order.
	//
	// If the workspace uses other workspaces, these used workspaces (but not the types from them) also returned.
	Types() []IType

	// Returns used workspaces in alphabetic order.
	//
	// Only direct used workspaces are enumerated.
	UsedWorkspaces() []IWorkspace
}

type IWorkspaceBuilder interface {
	ITypeBuilder
	IWithAbstractBuilder

	ITagsBuilder

	IDataTypesBuilder

	IGDocsBuilder
	ICDocsBuilder
	IWDocsBuilder
	IODocsBuilder
	IObjectsBuilder

	IViewsBuilder

	ICommandsBuilder
	IQueriesBuilder

	IProjectorsBuilder
	IJobsBuilder

	IRolesBuilder
	IACLBuilder

	IRatesBuilder
	ILimitsBuilder

	// Sets workspace ancestors.
	//
	// Ancestors are used to inherit types from other workspaces.
	// Circular inheritance is not allowed.
	// If no ancestors are set, workspace inherits types from `sys.Workspace`.
	//
	// # Panics:
	//	- if ancestor workspace is not found,
	//	- if ancestor workspace inherits from this workspace.
	SetAncestors(QName, ...QName) IWorkspaceBuilder

	// Sets descriptor.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	//	- if name is not CDoc
	SetDescriptor(QName) IWorkspaceBuilder

	// Adds used workspace.
	//
	// # Panics:
	//	- if used workspace is not found,
	//	- if workspace already used.
	UseWorkspace(QName, ...QName) IWorkspaceBuilder

	// Returns workspace definition while building.
	//
	// Can be called before or after all workspace entities added.
	// Does not validate workspace definition, may be invalid.
	Workspace() IWorkspace
}

type IWithWorkspaces interface {
	// Returns workspace by name.
	//
	// Returns nil if not found.
	Workspace(QName) IWorkspace

	// Returns workspace by descriptor.
	//
	// Returns nil if not found.
	WorkspaceByDescriptor(QName) IWorkspace

	// Enumerates all application workspaces.
	//
	// Workspaces are enumerated in alphabetical order by QName
	Workspaces() []IWorkspace
}

type IWorkspacesBuilder interface {
	// Adds new workspace.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWorkspace(QName) IWorkspaceBuilder

	// Returns builder for altering existing workspace.
	//
	// # Panics:
	//	 - if workspace with name does not exist.
	AlterWorkspace(QName) IWorkspaceBuilder
}
