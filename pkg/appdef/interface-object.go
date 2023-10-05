/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Object type.
//
// Ref. to object.go for implementation
type IObject interface {
	IStructure

	// Unwanted type assertion stub
	IsObject() bool
}

type IObjectBuilder interface {
	IObject
	IStructureBuilder
}

// Element type.
//
// Ref. to object.go for implementation
type IElement interface {
	IStructure

	// Unwanted type assertion stub
	IsElement() bool
}

type IElementBuilder interface {
	IElement
	IStructureBuilder
}
