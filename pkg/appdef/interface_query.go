/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Query is a function that returns data from system state.
type IQuery interface {
	IFunction
}

type IQueryBuilder interface {
	IFunctionBuilder
}

type IQueriesBuilder interface {
	// Adds new query.
	//
	// # Panics:
	//   - if name is empty (appdef.NullQName),
	//   - if name is invalid,
	//   - if type with name already exists.
	AddQuery(QName) IQueryBuilder
}
