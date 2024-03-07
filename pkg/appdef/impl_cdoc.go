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
func newCDoc(app *appDef, name QName) *cDoc {
	d := &cDoc{
		singleton: makeSingleton(app, name, TypeKind_CDoc),
	}
	app.appendType(d)
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

func newCRecord(app *appDef, name QName) *cRecord {
	r := &cRecord{
		containedRecord: makeContainedRecord(app, name, TypeKind_CRecord),
	}
	app.appendType(r)
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
