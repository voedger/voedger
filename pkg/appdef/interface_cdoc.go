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
	ICDoc
	ISingletonBuilder
}

// Configuration document record.
type ICRecord interface {
	IContainedRecord

	// Unwanted type assertion stub
	isCRecord()
}

type ICRecordBuilder interface {
	ICRecord
	IContainedRecordBuilder
}
