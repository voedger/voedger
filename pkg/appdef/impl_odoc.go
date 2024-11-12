/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IODoc
type oDoc struct {
	doc
}

func newODoc(app *appDef, ws *workspace, name QName) *oDoc {
	d := &oDoc{}
	d.doc = makeDoc(app, ws, name, TypeKind_ODoc)
	ws.appendType(d)
	return d
}

func (d *oDoc) isODoc() {}

// # Implements:
//   - IODocBuilder
type oDocBuilder struct {
	docBuilder
	*oDoc
}

func newODocBuilder(oDoc *oDoc) *oDocBuilder {
	return &oDocBuilder{
		docBuilder: makeDocBuilder(&oDoc.doc),
		oDoc:       oDoc,
	}
}

// # Implements:
//	- IORecord
type oRecord struct {
	containedRecord
}

func newORecord(app *appDef, ws *workspace, name QName) *oRecord {
	r := &oRecord{}
	r.containedRecord = makeContainedRecord(app, ws, name, TypeKind_ORecord)
	ws.appendType(r)
	return r
}

func (r oRecord) isORecord() {}

// # Implements:
//   - IORecordBuilder
type oRecordBuilder struct {
	containedRecordBuilder
	*oRecord
}

func newORecordBuilder(oRecord *oRecord) *oRecordBuilder {
	return &oRecordBuilder{
		containedRecordBuilder: makeContainedRecordBuilder(&oRecord.containedRecord),
		oRecord:                oRecord,
	}
}
