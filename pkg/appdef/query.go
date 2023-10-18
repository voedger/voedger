/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IQuery
//   - IQueryBuilder
type query struct {
	function
}

func newQuery(app *appDef, name QName) *query {
	q := &query{}
	q.function = makeFunc(app, name, TypeKind_Query, q)
	app.appendType(q)
	return q
}

func (q *query) isQuery() {}
