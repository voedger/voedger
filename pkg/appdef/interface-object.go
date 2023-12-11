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
	isObject()
}

type IObjectBuilder interface {
	IObject
	IStructureBuilder
}
