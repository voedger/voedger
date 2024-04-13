/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workspace is a set of types.
type IWorkspace interface {
	IType
	IWithAbstract
	IWithTypes

	// Workspace descriptor document.
	// See [#466](https://github.com/voedger/voedger/issues/466)
	//
	// Descriptor is CDoc document.
	// If the Descriptor is an abstract document, the workspace must also be abstract.
	Descriptor() QName
}

type IWorkspaceBuilder interface {
	ITypeBuilder
	IWithAbstractBuilder

	// Adds (includes) type to workspace. Type must be defined for application before.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	AddType(QName) IWorkspaceBuilder

	// Sets descriptor.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	//	- if name is not CDoc
	SetDescriptor(QName) IWorkspaceBuilder

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
	Workspaces(func(IWorkspace))
}

type IWorkspacesBuilder interface {
	// Adds new workspace.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddWorkspace(QName) IWorkspaceBuilder
}
