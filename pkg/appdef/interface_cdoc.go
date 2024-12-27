/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Configuration document.
type ICDoc interface {
	ISingleton
}

type ICDocBuilder interface {
	ISingletonBuilder
}

// Configuration document record.
type ICRecord interface {
	IContainedRecord
}

type ICRecordBuilder interface {
	IContainedRecordBuilder
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
