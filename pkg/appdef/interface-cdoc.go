/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Configuration document. DefKind() is DefKind_CDoc.
type ICDoc interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract

	// Returns is singleton
	Singleton() bool
}

type ICDocBuilder interface {
	ICDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder

	// Sets CDoc singleton
	SetSingleton()
}

// Configuration document record. DefKind() is DefKind_CRecord.
//
// Ref. to cdoc.go for implementation
type ICRecord interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type ICRecordBuilder interface {
	ICRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}
