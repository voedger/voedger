/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IWDoc, IWDocBuilder
type wDoc struct {
	def
	fields
	containers
	uniques
}

func newWDoc(app *appDef, name QName) *wDoc {
	doc := &wDoc{
		def: makeDef(app, name, DefKind_WDoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	doc.uniques = makeUniques(doc)
	app.appendDef(doc)
	return doc
}

// # Implements:
//   - IWRecord, IWRecordBuilder
type wRecord struct {
	def
	fields
	containers
	uniques
}

func newWRecord(app *appDef, name QName) *wRecord {
	rec := &wRecord{
		def: makeDef(app, name, DefKind_WRecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	rec.uniques = makeUniques(rec)
	app.appendDef(rec)
	return rec
}
