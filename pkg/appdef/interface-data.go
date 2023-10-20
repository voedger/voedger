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

	// Constraints for data type.
	Constraints() IDataConstraints
}

type IDataBuilder interface {
	ITypeBuilder
	IData

	// Add data constraint.
	//
	// # Panics:
	//	 - if kind is not supported for data type.
	AddConstraints(c ...IDataConstraint) IDataBuilder
}

// Data constraint kind enumeration.
//
// Ref. data-constraint-kind.go for constants and methods.
type DataConstraintKind uint8

// Data type constraints interface.
//
// Ref. data-constraint.go for implementation.
type IDataConstraints interface {
	// Returns constraints count.
	Count() int

	// Returns constraint by kind.
	//
	// Returns nil if constraint is not exists.
	Constraint(kind DataConstraintKind) IDataConstraint
}

// Data constraint interface.
//
// Ref. data-constraint.go for constraints constructors.
type IDataConstraint interface {
	IComment

	// Returns constraint kind.
	Kind() DataConstraintKind

	// Returns constraint value.
	Value() any
}
