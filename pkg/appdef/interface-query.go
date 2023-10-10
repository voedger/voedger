/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Query is a function
//
// Ref. to query.go for implementation
type IQuery interface {
	IFunc

	// Unwanted type assertion stub
	isQuery()
}

type IQueryBuilder interface {
	IQuery
	IFuncBuilder
}
