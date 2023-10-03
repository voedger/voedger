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
}

type IRecordBuilder interface {
	IRecord
	IStructureBuilder
}
