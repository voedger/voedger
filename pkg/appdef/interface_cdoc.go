/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Configuration document.
type ICDoc interface {
	ISingleton

	// Unwanted type assertion stub
	isCDoc()
}

type ICDocBuilder interface {
	ISingletonBuilder
}

// Configuration document record.
type ICRecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	isCRecord()
}

type ICRecordBuilder interface {
	IContainedRecordBuilder
}

type IWithCDocs interface {
	// Return CDoc by name.
	//
	// Returns nil if not found.
	CDoc(name QName) ICDoc

	// Return CRecord by name.
	//
	// Returns nil if not found.
	CRecord(name QName) ICRecord

	// Enumerates all application configuration documents
	//
	// Configuration documents are enumerated in alphabetical order by QName
	CDocs(func(ICDoc))

	// Enumerates all application configuration records
	//
	// Configuration records are enumerated in alphabetical order by QName
	CRecords(func(ICRecord))
}

type ICDocsBuilder interface {
	// Adds new CDoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCDoc(name QName) ICDocBuilder

	// Adds new CRecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddCRecord(name QName) ICRecordBuilder
}
