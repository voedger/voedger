/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Workflow document.
type IWDoc interface {
	IDoc
}

type IWDocBuilder interface {
	IWDoc
	IDocBuilder
}

// Workflow document record.
//
// Ref. to wdoc.go for implementation
type IWRecord interface {
	IRecord
}

type IWRecordBuilder interface {
	IWRecord
	IRecordBuilder
}
