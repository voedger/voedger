/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.IQuery
type Query struct {
	Function
}

func NewQuery(ws appdef.IWorkspace, name appdef.QName) *Query {
	return &Query{Function: MakeFunc(ws, name, appdef.TypeKind_Query)}
}

// # Supports:
//   - appdef.IQueryBuilder
type QueryBuilder struct {
	FunctionBuilder
	*Query
}

func NewQueryBuilder(q *Query) *QueryBuilder {
	return &QueryBuilder{
		FunctionBuilder: MakeFunctionBuilder(&q.Function),
		Query:           q,
	}
}
