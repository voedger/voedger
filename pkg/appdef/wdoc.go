/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IWDoc, IWDocBuilder
type wDoc struct {
	typ
	comment
	fields
	containers
	uniques
	withAbstract
}

func newWDoc(app *appDef, name QName) *wDoc {
	doc := &wDoc{
		typ: makeType(app, name, TypeKind_WDoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	doc.uniques = makeUniques(doc)
	app.appendType(doc)
	return doc
}

// # Implements:
//   - IWRecord, IWRecordBuilder
type wRecord struct {
	typ
	comment
	fields
	containers
	uniques
	withAbstract
}

func newWRecord(app *appDef, name QName) *wRecord {
	rec := &wRecord{
		typ: makeType(app, name, TypeKind_WRecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	rec.uniques = makeUniques(rec)
	app.appendType(rec)
	return rec
}
