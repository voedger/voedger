/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IGDoc
type gDoc struct {
	doc
}

func newGDoc(app *appDef, ws *workspace, name QName) *gDoc {
	d := &gDoc{}
	d.doc = makeDoc(app, ws, name, TypeKind_GDoc)
	ws.appendType(d)
	return d
}

func (d gDoc) isGDoc() {}

// # Implements:
//   - IGDocBuilder
type gDocBuilder struct {
	docBuilder
	*gDoc
}

func newGDocBuilder(gDoc *gDoc) *gDocBuilder {
	return &gDocBuilder{
		docBuilder: makeDocBuilder(&gDoc.doc),
		gDoc:       gDoc,
	}
}

// # Implements:
//   - IGRecord
type gRecord struct {
	containedRecord
}

func (r gRecord) isGRecord() {}

func newGRecord(app *appDef, ws *workspace, name QName) *gRecord {
	r := &gRecord{}
	r.containedRecord = makeContainedRecord(app, ws, name, TypeKind_GRecord)
	ws.appendType(r)
	return r
}

// # Implements:
//   - IGRecordBuilder
type gRecordBuilder struct {
	containedRecordBuilder
	*gRecord
}

func newGRecordBuilder(gRecord *gRecord) *gRecordBuilder {
	return &gRecordBuilder{
		containedRecordBuilder: makeContainedRecordBuilder(&gRecord.containedRecord),
		gRecord:                gRecord,
	}
}
