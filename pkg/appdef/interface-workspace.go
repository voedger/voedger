/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workspace
//
// Ref. to workspace.go for implementation
type IWorkspace interface {
	IDef
	IComment
	IWithAbstract

	// Returns definition by name.
	//
	// Nil is returned if not found
	Def(QName) IDef

	// Enumerates all workspace definitions
	Defs(func(IDef))

	// Workspace descriptor document.
	// See [#466](https://github.com/voedger/voedger/issues/466)
	//
	// Descriptor is CDoc document.
	// If the Descriptor is an abstract document, the workspace must also be abstract.
	Descriptor() QName
}

type IWorkspaceBuilder interface {
	IWorkspace
	ICommentBuilder
	IWithAbstractBuilder

	// Adds definition to workspace. Definition must be defined for application before.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	AddDef(QName) IWorkspaceBuilder

	// Sets descriptor.
	//
	// # Panics:
	//	- if name is empty
	//	- if name is not defined for application
	//	- if name is not CDoc
	SetDescriptor(QName) IWorkspaceBuilder
}
