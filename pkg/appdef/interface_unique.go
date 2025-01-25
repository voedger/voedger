/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Final structures with uniques are:
// - TypeKind_GDoc and TypeKind_GRecord,
// - TypeKind_CDoc and TypeKind_CRecord,
// - TypeKind_WDoc and TypeKind_WRecord
type IWithUniques interface {
	// Return unique by qualified name.
	//
	// Returns nil if not unique found
	UniqueByName(QName) IUnique

	// Return uniques count
	UniqueCount() int

	// All uniques as map. Key is unique name. Value is unique.
	Uniques() map[QName]IUnique

	// Returns single field unique.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	UniqueField() IField
}

type IUniquesBuilder interface {
	// Adds new unique with specified name and fields.
	//
	// # Panics:
	//   - if unique name is empty,
	//   - if unique name is invalid,
	//   - if name is already exists,
	//   - if structured type kind is not supports uniques,
	//   - if fields list is empty,
	//   - if fields has duplicates,
	//   - if fields is already exists or overlaps with an existing unique,
	//   - if some field not found.
	AddUnique(name QName, fields []FieldName, comment ...string) IUniquesBuilder

	// Sets single field unique.
	// Calling SetUniqueField again changes unique field. If specified name is empty, then clears unique field.
	//
	// This is old-style unique support. See [issue #173](https://github.com/voedger/voedger/issues/173)
	//
	// # Panics:
	//   - if field name is invalid,
	//   - if field not found,
	//   - if field is not required.
	SetUniqueField(FieldName) IUniquesBuilder
}

// Describe single unique for structure.
type IUnique interface {
	IWithComments

	// Returns qualified name of unique.
	Name() QName

	// Returns unique fields list in alphabetically order
	Fields() []IField
}
