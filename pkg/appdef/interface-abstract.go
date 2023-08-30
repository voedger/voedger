/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// See [Issue #524](https://github.com/voedger/voedger/issues/524)
// Definition can be abstract:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element
//	- DefKind_Workspace
//
// Ref to abstract.go for implementation
type IWithAbstract interface {
	// Returns is definition abstract
	Abstract() bool
}

type IWithAbstractBuilder interface {
	IWithAbstract

	// Makes definition abstract
	SetAbstract()
}
