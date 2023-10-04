/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IGDoc, IGDocBuilder
type gDoc struct {
	doc
}

func newGDoc(app *appDef, name QName) *gDoc {
	d := &gDoc{}
	d.doc = makeDoc(app, name, TypeKind_GDoc, d)
	app.appendType(d)
	return d
}

// # Implements:
//   - IGRecord, IGRecordBuilder
type gRecord struct {
	record
}

func newGRecord(app *appDef, name QName) *gRecord {
	r := &gRecord{}
	r.record = makeRecord(app, name, TypeKind_GRecord, r)
	app.appendType(r)
	return r
}
