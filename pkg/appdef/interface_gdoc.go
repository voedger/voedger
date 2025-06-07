/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document.
type IGDoc interface {
	IDoc

	// Unwanted type assertion stub
	IsGDoc()
}

type IGDocBuilder interface {
	IDocBuilder
}

// Global document record.
type IGRecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	IsGRecord()
}

type IGRecordBuilder interface {
	IContainedRecordBuilder
}

type IGDocsBuilder interface {
	// Adds new GDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddGDoc(QName) IGDocBuilder

	// Adds new GRecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddGRecord(QName) IGRecordBuilder
}
