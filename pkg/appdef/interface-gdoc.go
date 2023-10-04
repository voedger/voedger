/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Global document
type IGDoc interface {
	IDoc
}

type IGDocBuilder interface {
	IGDoc
	IDocBuilder
}

// Global document record
//
// Ref. to gdoc.go for implementation
type IGRecord interface {
	IRecord
}

type IGRecordBuilder interface {
	IGRecord
	IRecordBuilder
}
