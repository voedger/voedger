/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workflow document.
type IWDoc interface {
	ISingleton

	// Unwanted type assertion stub
	isWDoc()
}

type IWDocBuilder interface {
	ISingletonBuilder
}

// Workflow document record.
type IWRecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	isWRecord()
}

type IWRecordBuilder interface {
	IContainedRecordBuilder
}
