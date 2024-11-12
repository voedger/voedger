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
	IStructureBuilder
}

type IObjectsBuilder interface {
	// Adds new Object type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddObject(name QName) IObjectBuilder
}
