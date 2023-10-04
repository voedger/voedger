/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Operation document.
//
// Ref. to odoc.go for implementation
type IODoc interface {
	IType
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

// Operation document record.
//
// Ref. to odoc.go for implementation
type IORecord interface {
	IType
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
