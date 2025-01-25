/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Numeric with OccursUnbounded value.
//
// Ref. occurs.go for constants and methods
type Occurs uint16

const (
	Occurs_Unbounded    = Occurs(0xffff)
	Occurs_UnboundedStr = "unbounded"
)

// Final structures with containers are:
//	- TypeKind_GDoc and TypeKind_GRecord,
//	- TypeKind_CDoc and TypeKind_CRecord,
//	- TypeKind_ODoc and TypeKind_CRecord,
//	- TypeKind_WDoc and TypeKind_WRecord,
//	- TypeKind_Object,
type IWithContainers interface {
	// Finds container by name.
	//
	// Returns nil if not found.
	Container(name string) IContainer

	// Returns containers count
	ContainerCount() int

	// All containers in add order.
	Containers() []IContainer
}

type IContainersBuilder interface {
	// Adds container specified name and occurs.
	//
	// # Panics:
	//   - if name is empty,
	//   - if name is invalid,
	//   - if container with name already exists,
	//   - if type name is empty,
	//   - if invalid occurrences,
	//   - if container type kind is not compatible with parent type kind.
	AddContainer(name string, typeName QName, minimum, maximum Occurs, comment ...string) IContainersBuilder
}

// Describes single inclusion of child in parent.
type IContainer interface {
	IWithComments

	// Returns name of container
	Name() string

	// Returns type name of included in container child
	QName() QName

	// Returns structure type of included in container child.
	//
	// Returns nil if not found
	Type() IStructure

	// Returns minimum occurs of child
	MinOccurs() Occurs

	// Returns maximum occurs of child
	MaxOccurs() Occurs
}
