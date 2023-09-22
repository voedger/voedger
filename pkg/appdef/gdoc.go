/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//   - IGDoc, IGDocBuilder
type gDoc struct {
	typ
	comment
	fields
	containers
	uniques
	withAbstract
}

func newGDoc(app *appDef, name QName) *gDoc {
	doc := &gDoc{
		typ: makeType(app, name, TypeKind_GDoc),
	}
	doc.fields = makeFields(doc)
	doc.containers = makeContainers(doc)
	doc.uniques = makeUniques(doc)
	app.appendType(doc)
	return doc
}

// # Implements:
//   - IGRecord, IGRecordBuilder
type gRecord struct {
	typ
	comment
	fields
	containers
	uniques
	withAbstract
}

func newGRecord(app *appDef, name QName) *gRecord {
	rec := &gRecord{
		typ: makeType(app, name, TypeKind_GRecord),
	}
	rec.fields = makeFields(rec)
	rec.containers = makeContainers(rec)
	rec.uniques = makeUniques(rec)
	app.appendType(rec)
	return rec
}
