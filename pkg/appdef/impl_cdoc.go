/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - ICDoc, ICDocBuilder
type cDoc struct {
	singleton
}

// Creates a new CDoc
func newCDoc(app *appDef, name QName) *cDoc {
	d := &cDoc{}
	d.singleton = makeSingleton(app, name, TypeKind_CDoc, d)
	app.appendType(d)
	return d
}

func (d *cDoc) isCDoc() {}

// # Implements:
//   - ICRecord, ICRecordBuilder
type cRecord struct {
	containedRecord
}

func newCRecord(app *appDef, name QName) *cRecord {
	r := &cRecord{}
	r.containedRecord = makeContainedRecord(app, name, TypeKind_CRecord, r)
	app.appendType(r)
	return r
}

func (r cRecord) isCRecord() {}
