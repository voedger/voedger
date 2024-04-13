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

type IWithObjects interface {
	// Return Object by name.
	//
	// Returns nil if not found.
	Object(name QName) IObject

	// Enumerates all application objects
	//
	// Objects are enumerated in alphabetical order by QName
	Objects(func(IObject))
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
