/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IGDoc, IGDocBuilder
type gDoc struct {
	def
	fields
	containers
	uniques
	withAbstract
}

func newGDoc(app *appDef, name QName) *gDoc {
	doc := &gDoc{
		def: makeDef(app, name, DefKind_GDoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	doc.uniques = makeUniques(doc)
	app.appendDef(doc)
	return doc
}

// # Implements:
//   - IGRecord, IGRecordBuilder
type gRecord struct {
	def
	fields
	containers
	uniques
	withAbstract
}

func newGRecord(app *appDef, name QName) *gRecord {
	rec := &gRecord{
		def: makeDef(app, name, DefKind_GRecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	rec.uniques = makeUniques(rec)
	app.appendDef(rec)
	return rec
}
