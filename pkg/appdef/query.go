/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - IQuery & IQueryBuilder
type query struct {
	def
	comment
	arg, res objRef
	ext      extension
}

func newQuery(app *appDef, name QName) *query {
	q := &query{
		def: makeDef(app, name, DefKind_Query),
	}
	app.appendDef(q)
	return q
}

func (q *query) Arg() IObject {
	return q.arg.object(q.app)
}

func (q *query) Extension() IExtension {
	return &q.ext
}

func (q *query) Result() IObject {
	return q.res.object(q.app)
}

func (q *query) SetArg(name QName) IQueryBuilder {
	q.arg.setName(name)
	return q
}

func (q *query) SetResult(name QName) IQueryBuilder {
	q.res.setName(name)
	return q
}

func (q *query) SetExtension(name string, engine ExtensionEngineKind) IQueryBuilder {
	if name == "" {
		panic(fmt.Errorf("%v: extension name is empty: %w", q.QName(), ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: extension name «%s» is not valid: %w", q.QName(), name, err))
	}
	q.ext.name = name
	q.ext.engine = engine
	return q
}

// validates query
func (q *query) Validate() (err error) {
	if q.arg.name != NullQName {
		if q.arg.object(q.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: argument definition «%v» is not found: %w", q.QName(), q.arg.name, ErrNameNotFound))
		}
	}

	if q.res.name != NullQName {
		if q.res.object(q.app) == nil {
			err = errors.Join(err, fmt.Errorf("%v: query result definition «%v» is not found: %w", q.QName(), q.res.name, ErrNameNotFound))
		}
	}

	if q.Extension().Name() == "" {
		err = errors.Join(err, fmt.Errorf("%v: query extension name is missed: %w", q.QName(), ErrNameMissed))
	}

	if q.Extension().Engine() == ExtensionEngineKind_null {
		err = errors.Join(err, fmt.Errorf("%v: query extension engine is missed: %w", q.QName(), ErrExtensionEngineKindMissed))
	}

	return err
}
