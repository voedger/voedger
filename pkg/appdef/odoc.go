/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IODoc, IODocBuilder
type oDoc struct {
	typ
	comment
	fields
	containers
	withAbstract
}

func newODoc(app *appDef, name QName) *oDoc {
	doc := &oDoc{
		typ: makeType(app, name, TypeKind_ODoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	app.appendType(doc)
	return doc
}

// # Implements:
//   - IORecord, IORecordBuilder
type oRecord struct {
	typ
	comment
	fields
	containers
	withAbstract
}

func newORecord(app *appDef, name QName) *oRecord {
	rec := &oRecord{
		typ: makeType(app, name, TypeKind_ORecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	app.appendType(rec)
	return rec
}
