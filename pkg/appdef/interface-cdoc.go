/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Configuration document
type ICDoc interface {
	IDoc

	// Returns is singleton
	Singleton() bool
}

type ICDocBuilder interface {
	ICDoc
	IDocBuilder

	// Sets CDoc singleton
	SetSingleton()
}

// Configuration document record.
//
// Ref. to cdoc.go for implementation
type ICRecord interface {
	IRecord

	// Unwanted type assertion stub
	isCRecord()
}

type ICRecordBuilder interface {
	ICRecord
	IRecordBuilder
}
