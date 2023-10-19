/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Data type interface.
//
// Describe simple types, like string, number, date, etc.
//
// Ref. to data.go for implementation
type IData interface {
	IType

	// Returns is data type is system.
	IsSystem() bool

	// Ref. to data-kind.go for details
	DataKind() DataKind

	// Ancestor	type.
	//
	// All user types should have ancestor. System types may has no ancestor.
	Ancestor() IData
}

type IDataBuilder interface {
	ITypeBuilder
	IData
}
