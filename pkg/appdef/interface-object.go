/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Object definition. DefKind() is DefKind_Object.
//
// Ref. to object.go for implementation
type IObject interface {
	IDef
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

// Element definition. DefKind() is DefKind_Element.
//
// Ref. to object.go for implementation
type IElement interface {
	IDef
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
