/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IODoc, IODocBuilder
type oDoc struct {
	doc
}

func newODoc(app *appDef, name QName) *oDoc {
	d := &oDoc{}
	d.doc = makeDoc(app, name, TypeKind_ODoc, d)
	app.appendType(d)
	return d
}

func (d *oDoc) IsODoc() bool { return true }

// # Implements:
//   - IORecord
//	-	IORecordBuilder
type oRecord struct {
	record
}

func newORecord(app *appDef, name QName) *oRecord {
	r := &oRecord{}
	r.record = makeRecord(app, name, TypeKind_ORecord, r)
	app.appendType(r)
	return r
}

func (r oRecord) IsORecord() bool { return true }
