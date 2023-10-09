/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Numeric with OccursUnbounded value.
//
// Ref. occurs.go for constants and methods
type Occurs uint16

// Types with containers:
//	- TypeKind_GDoc and TypeKind_GRecord,
//	- TypeKind_CDoc and TypeKind_CRecord,
//	- TypeKind_ODoc and TypeKind_CRecord,
//	- TypeKind_WDoc and TypeKind_WRecord,
//	- TypeKind_Object and TypeKind_Element,
//
// Ref. to container.go for implementation
type IContainers interface {
	// Finds container by name.
	//
	// Returns nil if not found.
	Container(name string) IContainer

	// Returns containers count
	ContainerCount() int

	// Enumerates all containers in add order.
	Containers(func(IContainer))
}

type IContainersBuilder interface {
	IContainers

	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if container with name already exists,
	//   - if type name is empty,
	//   - if invalid occurrences,
	//   - if container type kind is not compatible with parent type kind.
	AddContainer(name string, typeName QName, min, max Occurs, comment ...string) IContainersBuilder
}

// Describes single inclusion of child in parent.
//
// Ref to container.go for implementation
type IContainer interface {
	IComment

	// Returns name of container
	Name() string

	// Returns type name of container
	QName() QName

	// Returns container type.
	//
	// Returns nil if not found
	Type() IType

	// Returns minimum occurs
	MinOccurs() Occurs

	// Returns maximum occurs
	MaxOccurs() Occurs
}
