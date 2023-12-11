/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document.
//
// Ref. to gdoc.go for implementation
type IGDoc interface {
	IDoc

	// unwanted type assertion stub
	isGDoc()
}

type IGDocBuilder interface {
	IGDoc
	IDocBuilder
}

// Global document record.
//
// Ref. to gdoc.go for implementation
type IGRecord interface {
	IContainedRecord

	// unwanted type assertion stub
	isGRecord()
}

type IGRecordBuilder interface {
	IGRecord
	IContainedRecordBuilder
}
