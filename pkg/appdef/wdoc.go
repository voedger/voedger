/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IWDoc, IWDocBuilder
type wDoc struct {
	doc
}

func newWDoc(app *appDef, name QName) *wDoc {
	d := &wDoc{}
	d.doc = makeDoc(app, name, TypeKind_WDoc, d)
	d.makeSysFields()
	app.appendType(d)
	return d
}

// # Implements:
//   - IWRecord, IWRecordBuilder
type wRecord struct {
	record
}

func newWRecord(app *appDef, name QName) *wRecord {
	r := &wRecord{}
	r.record = makeRecord(app, name, TypeKind_WRecord, r)
	r.makeSysFields()
	app.appendType(r)
	return r
}
