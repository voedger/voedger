/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workflow document.
type IWDoc interface {
	IType
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IWDocBuilder interface {
	IWDoc
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Workflow document record.
//
// Ref. to wdoc.go for implementation
type IWRecord interface {
	IType
	IComment
	IFields
	IContainers
	IUniques
	IWithAbstract
}

type IWRecordBuilder interface {
	IWRecord
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}
