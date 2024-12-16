/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - ICDoc
type cDoc struct {
	singleton
}

// Creates a new CDoc
func newCDoc(app *appDef, ws *workspace, name QName) *cDoc {
	d := &cDoc{
		singleton: makeSingleton(app, ws, name, TypeKind_CDoc),
	}
	ws.appendType(d)
	return d
}

func (d *cDoc) isCDoc() {}

// # Implements:
//   - ICDocBuilder
type cDocBuilder struct {
	singletonBuilder
	*cDoc
}

func newCDocBuilder(cDoc *cDoc) *cDocBuilder {
	return &cDocBuilder{
		singletonBuilder: makeSingletonBuilder(&cDoc.singleton),
		cDoc:             cDoc,
	}
}

// # Implements:
//   - ICRecord
type cRecord struct {
	containedRecord
}

func newCRecord(app *appDef, ws *workspace, name QName) *cRecord {
	r := &cRecord{
		containedRecord: makeContainedRecord(app, ws, name, TypeKind_CRecord),
	}
	ws.appendType(r)
	return r
}

func (r cRecord) isCRecord() {}

// # Implements:
//   - ICRecordBuilder
type cRecordBuilder struct {
	containedRecordBuilder
	*cRecord
}

func newCRecordBuilder(cRecord *cRecord) *cRecordBuilder {
	return &cRecordBuilder{
		containedRecordBuilder: makeContainedRecordBuilder(&cRecord.containedRecord),
		cRecord:                cRecord,
	}
}
