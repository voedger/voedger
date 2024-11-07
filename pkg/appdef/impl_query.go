/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IQuery
type query struct {
	function
}

func newQuery(app *appDef, ws *workspace, name QName) *query {
	q := &query{}
	q.function = makeFunc(app, ws, name, TypeKind_Query)
	ws.appendType(q)
	return q
}

func (q *query) isQuery() {}

// # Implements:
//   - IQueryBuilder
type queryBuilder struct {
	functionBuilder
	*query
}

func newQueryBuilder(q *query) *queryBuilder {
	return &queryBuilder{
		functionBuilder: makeFunctionBuilder(&q.function),
		query:           q,
	}
}
