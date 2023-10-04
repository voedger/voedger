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
}

type IElementBuilder interface {
	IElement
	IStructureBuilder
}
