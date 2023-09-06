/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document. DefKind() is DefKind_GDoc.
type IGDoc interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IGDocBuilder interface {
	IGDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Global document record. DefKind() is DefKind_GRecord.
//
// Ref. to gdoc.go for implementation
type IGRecord interface {
	IDef
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IGRecordBuilder interface {
	IGRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}
