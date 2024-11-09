/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Operation document.
type IODoc interface {
	IDoc

	// Unwanted type assertion stub
	isODoc()
}

type IODocBuilder interface {
	IDocBuilder
}

// Operation document record.
type IORecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	isORecord()
}

type IORecordBuilder interface {
	IContainedRecordBuilder
}

type IODocsBuilder interface {
	// Adds new ODoc type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddODoc(name QName) IODocBuilder

	// Adds new ORecord type with specified name.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddORecord(name QName) IORecordBuilder
}
