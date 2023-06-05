/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IODoc, IODocBuilder
type oDoc struct {
	def
	fields
	containers
}

func newODoc(app *appDef, name QName) *oDoc {
	doc := &oDoc{
		def: makeDef(app, name, DefKind_ODoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	app.appendDef(doc)
	return doc
}

// # Implements:
//   - IORecord, IORecordBuilder
type oRecord struct {
	def
	fields
	containers
}

func newORecord(app *appDef, name QName) *oRecord {
	rec := &oRecord{
		def: makeDef(app, name, DefKind_ORecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	app.appendDef(rec)
	return rec
}
