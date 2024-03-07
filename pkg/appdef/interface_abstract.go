/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// See [Issue #524](https://github.com/voedger/voedger/issues/524).
// Final types can be abstract are:
//	- TypeKind_GDoc and TypeKind_GRecord,
//	- TypeKind_CDoc and TypeKind_CRecord,
//	- TypeKind_ODoc and TypeKind_CRecord,
//	- TypeKind_WDoc and TypeKind_WRecord,
//	- TypeKind_Object and TypeKind_Element
//	- TypeKind_Workspace
type IWithAbstract interface {
	// Returns is type abstract
	Abstract() bool
}

type IWithAbstractBuilder interface {
	// Makes type abstract
	SetAbstract()
}
