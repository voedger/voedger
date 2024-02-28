/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workflow document.
type IWDoc interface {
	IDoc

	// Unwanted type assertion stub
	isWDoc()
}

type IWDocBuilder interface {
	IWDoc
	IDocBuilder
}

// Workflow document record.
type IWRecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	isWRecord()
}

type IWRecordBuilder interface {
	IWRecord
	IContainedRecordBuilder
}
