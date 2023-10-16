/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// See [Issue #524](https://github.com/voedger/voedger/issues/524)
// Types can be abstract:
//	- TypeKind_GDoc and TypeKind_GRecord,
//	- TypeKind_CDoc and TypeKind_CRecord,
//	- TypeKind_ODoc and TypeKind_CRecord,
//	- TypeKind_WDoc and TypeKind_WRecord,
//	- TypeKind_Object and TypeKind_Element
//	- TypeKind_Workspace
//
// Ref to abstract.go for implementation
type IWithAbstract interface {
	// Returns is type abstract
	Abstract() bool
}

type IWithAbstractBuilder interface {
	IWithAbstract

	// Makes type abstract
	SetAbstract()
}
