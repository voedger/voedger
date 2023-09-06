/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - ICDoc, ICDocBuilder
type cDoc struct {
	def
	comment
	fields
	containers
	uniques
	withAbstract
	singleton bool
}

func newCDoc(app *appDef, name QName) *cDoc {
	doc := &cDoc{
		def: makeDef(app, name, DefKind_CDoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	doc.uniques = makeUniques(doc)
	app.appendDef(doc)
	return doc
}

func (d *cDoc) SetSingleton() {
	d.singleton = true
}

func (d *cDoc) Singleton() bool {
	return d.singleton
}

// # Implements:
//   - ICRecord, ICRecordBuilder
type cRecord struct {
	def
	comment
	fields
	containers
	uniques
	withAbstract
}

func newCRecord(app *appDef, name QName) *cRecord {
	rec := &cRecord{
		def: makeDef(app, name, DefKind_CRecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	rec.uniques = makeUniques(rec)
	app.appendDef(rec)
	return rec
}
