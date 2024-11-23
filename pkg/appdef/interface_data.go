/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Data kind enumeration.
//
// Ref. data-kind.go for constants and methods
type DataKind uint8

//go:generate stringer -type=DataKind -output=stringer_datakind.go

const (
	// null - no-value type. Returned when the requested type does not exist
	DataKind_null DataKind = iota

	DataKind_int32
	DataKind_int64
	DataKind_float32
	DataKind_float64
	DataKind_bytes
	DataKind_string
	DataKind_QName
	DataKind_bool

	DataKind_RecordID

	// Complex types

	DataKind_Record
	DataKind_Event

	DataKind_FakeLast
)

// Data type interface.
//
// Describe simple types, like string, number, date, etc.
type IData interface {
	IType

	DataKind() DataKind

	// Ancestor	type.
	//
	// All user types should have ancestor. System types may has no ancestor.
	Ancestor() IData

	// All data type constraints.
	//
	// To obtain all constraints include ancestor data types, pass true to withInherited parameter.
	Constraints(withInherited bool) map[ConstraintKind]IConstraint
}

type IDataBuilder interface {
	ITypeBuilder

	// Add data constraint.
	//
	// # Panics:
	//	 - if constraint is not compatible with data type.
	AddConstraints(c ...IConstraint) IDataBuilder
}

// Data types builder interface.
type IDataTypesBuilder interface {
	// Adds new data type with specified name and kind.
	//
	// If ancestor is not empty, then new data type inherits from.
	// If ancestor is empty, then new data type inherits from system data types with same data kind.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists,
	//   - if ancestor is not found,
	//	 - if ancestor is not data,
	//	 - if ancestor has different kind,
	//	 - if constraints are not compatible with data kind.
	AddData(name QName, kind DataKind, ancestor QName, constraints ...IConstraint) IDataBuilder
}

// Data constraint kind enumeration.
//
// Ref. data-constraint-kind.go for constants and methods.
type ConstraintKind uint8

//go:generate stringer -type=ConstraintKind -output=stringer_constraintkind.go

const (
	// null - no-value type. Returned when the requested kind does not exist
	ConstraintKind_null ConstraintKind = iota

	ConstraintKind_MinLen
	ConstraintKind_MaxLen
	ConstraintKind_Pattern

	ConstraintKind_MinIncl
	ConstraintKind_MinExcl
	ConstraintKind_MaxIncl
	ConstraintKind_MaxExcl

	ConstraintKind_Enum

	ConstraintKind_count
)

// Data constraint interface.
//
// Ref. data-constraint.go for constraints constructors and methods.
type IConstraint interface {
	IWithComments

	// Returns constraint kind.
	Kind() ConstraintKind

	// Returns constraint value.
	//
	// # Returns:
	//	- uint16 value for min/max length constraints,
	// 	- *regexp.Regexp value for pattern constraint,
	// 	- float64 value for min/max inclusive/exclusive constraints.
	//	- sorted slice with values for enumeration constraint.
	Value() any
}
