/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Object type.
//
// Ref. to object.go for implementation
type IObject interface {
	IType
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IObjectBuilder interface {
	IObject
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}

// Element type.
//
// Ref. to object.go for implementation
type IElement interface {
	IType
	IComment
	IFields
	IContainers
	IWithAbstract
}

type IElementBuilder interface {
	IElement
	ICommentBuilder
	IFieldsBuilder
	IContainersBuilder
	IWithAbstractBuilder
}
