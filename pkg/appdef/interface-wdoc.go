/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workflow document. DefKind() is DefKind_WDoc.
type IWDoc interface {
	IDef
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

// Workflow document record. DefKind() is DefKind_WRecord.
//
// Ref. to wdoc.go for implementation
type IWRecord interface {
	IDef
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
