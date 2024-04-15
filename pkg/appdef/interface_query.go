/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Query is a function that returns data from system state.
type IQuery interface {
	IFunction

	// Unwanted type assertion stub
	isQuery()
}

type IWithQueries interface {
	// Returns Query by name.
	//
	// Returns nil if not found.
	Query(QName) IQuery

	// Enumerates all queries
	//
	// Queries are enumerated in alphabetical order by QName
	Queries(func(IQuery))
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
