/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Structure is a type with fields, containers and uniques.
type IStructure interface {
	IType
	IWithFields
	IWithContainers
	IWithUniques
	IWithAbstract

	// Returns definition for «sys.QName» field
	SystemField_QName() IField
}

type IStructureBuilder interface {
	ITypeBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Record is a structure.
//
// Record has ID field.
type IRecord interface {
	IStructure

	// Returns definition for «sys.ID» field
	SystemField_ID() IField
}

type IRecordBuilder interface {
	IStructureBuilder
}

// Document is a record.
//
// Document can contains records.
type IDoc interface {
	IRecord
}

type IDocBuilder interface {
	IRecordBuilder
}

// Contained record is a record that has parent.
type IContainedRecord interface {
	IRecord

	// Returns definition for «sys.ParentID» field
	SystemField_ParentID() IField

	// Returns definition for «sys.Container» field
	SystemField_Container() IField
}

type IContainedRecordBuilder interface {
	IRecordBuilder
}
