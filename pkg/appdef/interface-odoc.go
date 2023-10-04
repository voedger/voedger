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
}

type IORecordBuilder interface {
	IORecord
	IRecordBuilder
}
