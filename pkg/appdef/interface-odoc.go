/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Operation document.
//
// Ref. to odoc.go for implementation
type IODoc interface {
	IDoc

	// Unwanted type assertion stub
	IsODoc() bool
}

type IODocBuilder interface {
	IODoc
	IDocBuilder
}

// Operation document record.
//
// Ref. to odoc.go for implementation
type IORecord interface {
	IRecord

	// Unwanted type assertion stub
	IsORecord() bool
}

type IORecordBuilder interface {
	IORecord
	IRecordBuilder
}
