/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Workspace
//
// Workspace is a set of types.
//
// Ref. to workspace.go for implementation
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
	IWorkspace
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
}
