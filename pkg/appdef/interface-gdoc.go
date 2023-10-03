/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document
type IGDoc interface {
	IType
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

// Global document record
//
// Ref. to gdoc.go for implementation
type IGRecord interface {
	IType
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
