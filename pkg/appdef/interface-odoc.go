/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Operation document. DefKind() is DefKind_ODoc.
type IODoc interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IODocBuilder interface {
	IODoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// Operation document record. DefKind() is DefKind_ORecord.
//
// Ref. to odoc.go for implementation
type IORecord interface {
	IDef
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IORecordBuilder interface {
	IORecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}
