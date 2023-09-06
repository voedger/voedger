/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Numeric with OccursUnbounded value.
//
// Ref. occurs.go for constants and methods
type Occurs uint16

// Definitions with containers:
//	- DefKind_GDoc and DefKind_GRecord,
//	- DefKind_CDoc and DefKind_CRecord,
//	- DefKind_ODoc and DefKind_CRecord,
//	- DefKind_WDoc and DefKind_WRecord,
//	- DefKind_Object and DefKind_Element,
//	- DefKind_ViewRecord and DefKind_ViewKey
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
	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if container with name already exists,
	//   - if definition name is empty,
	//   - if invalid occurrences,
	//   - if container definition kind is not compatible with parent definition kind.
	AddContainer(name string, def QName, min, max Occurs, comment ...string) IContainersBuilder
}

// Describes single inclusion of child definition in parent definition.
//
// Ref to container.go for implementation
type IContainer interface {
	IComment

	// Returns name of container
	Name() string

	// Returns definition name of container
	QName() QName

	// Returns container definition.
	//
	// Returns nil if not found
	Def() IDef

	// Returns minimum occurs
	MinOccurs() Occurs

	// Returns maximum occurs
	MaxOccurs() Occurs

	// Returns is container system
	IsSys() bool
}
