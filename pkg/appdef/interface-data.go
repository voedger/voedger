/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Data kind enumeration.
//
// Ref. data-kind.go for constants and methods
type DataKind uint8

// Data type interface.
//
// Describe simple types, like string, number, date, etc.
//
// Ref. to data.go for implementation
type IData interface {
	IType

	// Returns is data type is system.
	IsSystem() bool

	// Ref. to data-kind.go for details.
	DataKind() DataKind

	// Ancestor	type.
	//
	// All user types should have ancestor. System types may has no ancestor.
	Ancestor() IData

	// Enumerate data type constraints.
	//
	// Constraints are enumerated in kind order.
	//
	// Only this data type constraints are enumerated. To obtain all constraints,
	// include inherited ones, use DataConstraintsInherited().
	Constraints(func(IConstraint))
}

type IDataBuilder interface {
	ITypeBuilder
	IData

	// Add data constraint.
	//
	// # Panics:
	//	 - if kind is not supported for data type.
	AddConstraints(c ...IConstraint) IDataBuilder
}

// Data constraint kind enumeration.
//
// Ref. data-constraint-kind.go for constants and methods.
type ConstraintKind uint8

// Data constraint interface.
//
// Ref. data-constraint.go for constraints constructors.
type IConstraint interface {
	IComment

	// Returns constraint kind.
	Kind() ConstraintKind

	// Returns constraint value.
	//
	// # Returns:
	//	- uint16 value for min/max length constraints,
	// 	- *regexp.Regexp value for pattern constraint,
	// 	- float64 value for min/max inclusive/exclusive constraints.
	Value() any
}
