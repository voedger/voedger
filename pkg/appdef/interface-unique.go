/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Unique identifier type
type UniqueID uint32

// Types with uniques:
//	- TypeKind_GDoc and TypeKind_GRecord,
//	- TypeKind_CDoc and TypeKind_CRecord,
//	- TypeKind_WDoc and TypeKind_WRecord
//
// Ref. to unique.go for implementation
type IUniques interface {
	// Return unique by ID.
	//
	// Returns nil if not unique found
	UniqueByID(id UniqueID) IUnique

	// Return unique by name.
	//
	// Returns nil if not unique found
	UniqueByName(name string) IUnique

	// Return uniques count
	UniqueCount() int

	// Enumerates all uniques.
	Uniques(func(IUnique))

	// Returns single field unique.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	UniqueField() IField
}

type IUniquesBuilder interface {
	// Adds new unique with specified name and fields.
	// If name is omitted, then default name is used, e.g. `unique01`.
	//
	// # Panics:
	//   - if unique name is invalid,
	//   - if unique with name is already exists,
	//   - if structured type kind is not supports uniques,
	//   - if fields list is empty,
	//   - if fields has duplicates,
	//   - if fields is already exists or overlaps with an existing unique,
	//   - if some field not found.
	AddUnique(name string, fields []string, comment ...string) IUniquesBuilder

	// Sets single field unique.
	// Calling SetUniqueField again changes unique field. If specified name is empty, then clears unique field.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	//
	// # Panics:
	//   - if field name is invalid,
	//   - if field not found,
	//   - if field is not required.
	SetUniqueField(name string) IUniquesBuilder
}

// Describe single unique for structured type.
//
// Ref to unique.go for implementation
type IUnique interface {
	IComment

	// Returns parent type
	ParentType() IType

	// Returns name of unique.
	//
	// Name suitable for debugging or error messages. Unique identification provided by ID
	Name() string

	// Returns unique fields list. Fields are sorted alphabetically
	Fields() []IField

	// Unique identifier.
	//
	// Must be assigned during AppStruct construction by calling SetID(UniqueID)
	ID() UniqueID
}
