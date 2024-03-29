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
