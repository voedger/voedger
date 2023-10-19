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

	// - data type from which the user data type is inherits or
	// - nil for built-in sys types
	Ancestor() IData
}

type IDataBuilder interface {
	IData
}
