/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IWDoc
type wDoc struct {
	singleton
}

func newWDoc(app *appDef, ws *workspace, name QName) *wDoc {
	d := &wDoc{
		singleton: makeSingleton(app, ws, name, TypeKind_WDoc),
	}
	ws.appendType(d)
	return d
}

func (d *wDoc) isWDoc() {}

// # Implements:
//   - IWDocBuilder
type wDocBuilder struct {
	singletonBuilder
	*wDoc
}

func newWDocBuilder(wDoc *wDoc) *wDocBuilder {
	return &wDocBuilder{
		singletonBuilder: makeSingletonBuilder(&wDoc.singleton),
		wDoc:             wDoc,
	}
}

// # Implements:
//   - IWRecord
type wRecord struct {
	containedRecord
}

func newWRecord(app *appDef, ws *workspace, name QName) *wRecord {
	r := &wRecord{
		containedRecord: makeContainedRecord(app, ws, name, TypeKind_WRecord),
	}
	ws.appendType(r)
	return r
}

func (r wRecord) isWRecord() {}

// # Implements:
//   - IWRecordBuilder
type wRecordBuilder struct {
	containedRecordBuilder
	*wRecord
}

func newWRecordBuilder(wRecord *wRecord) *wRecordBuilder {
	return &wRecordBuilder{
		containedRecordBuilder: makeContainedRecordBuilder(&wRecord.containedRecord),
		wRecord:                wRecord,
	}
}
