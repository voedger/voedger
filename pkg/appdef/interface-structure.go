/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Structure is a type with fields, containers and uniques.
//
// Ref. to structure.go for implementation
type IStructure interface {
	IType
	IFields
	IContainers
	IUniques
	IWithAbstract

	// Returns definition for «sys.QName» field
	SystemField_QName() IField
}

type IStructureBuilder interface {
	IStructure
	ITypeBuilder
	IFieldsBuilder
	IContainersBuilder
	IUniquesBuilder
	IWithAbstractBuilder
}

// Document is a structure.
//
// Document can contains records.
type IDoc interface {
	IStructure

	// Returns definition for «sys.ID» field
	SystemField_ID() IField

	// Unwanted type assertion stub
	isDoc()
}

type IDocBuilder interface {
	IDoc
	IStructureBuilder
}

// Record is a structure.
//
// Record can contains child records.
type IRecord interface {
	IStructure

	// Returns definition for «sys.ID» field
	SystemField_ID() IField

	// Returns definition for «sys.ParentID» field
	SystemField_ParentID() IField

	// Returns definition for «sys.Container» field
	SystemField_Container() IField

	// Unwanted type assertion stub
	isRecord()
}

type IRecordBuilder interface {
	IRecord
	IStructureBuilder
}
