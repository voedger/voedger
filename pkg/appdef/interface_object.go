/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Object type.
type IObject interface {
	IStructure

	// Unwanted type assertion stub
	isObject()
}

type IObjectBuilder interface {
	IObject
	IStructureBuilder
}
