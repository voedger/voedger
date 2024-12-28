/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IQuery
type Query struct {
	Function
}

func NewQuery(ws appdef.IWorkspace, name appdef.QName) *Query {
	q := &Query{Function: MakeFunc(ws, name, appdef.TypeKind_Query)}
	types.Propagate(q)
	return q
}

func (Query) IsQuery() {}

// # Supports:
//   - appdef.IQueryBuilder
type QueryBuilder struct {
	FunctionBuilder
	q *Query
}

func NewQueryBuilder(q *Query) *QueryBuilder {
	return &QueryBuilder{
		FunctionBuilder: MakeFunctionBuilder(&q.Function),
		q:               q,
	}
}
