/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document.
type IGDoc interface {
	IDoc

	// unwanted type assertion stub
	isGDoc()
}

type IGDocBuilder interface {
	IDocBuilder
}

// Global document record.
type IGRecord interface {
	IContainedRecord

	// unwanted type assertion stub
	isGRecord()
}

type IGRecordBuilder interface {
	IContainedRecordBuilder
}

type IWithGDocs interface {
	// Return GDoc by name.
	//
	// Returns nil if not found.
	GDoc(QName) IGDoc

	// Enumerates all global documents
	//
	// Global documents are enumerated in alphabetical order by QName
	GDocs(func(IGDoc))

	// Return GRecord by name.
	//
	// Returns nil if not found.
	GRecord(QName) IGRecord

	// Enumerates all global records
	//
	// Global records are enumerated in alphabetical order by QName
	GRecords(func(IGRecord))
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
