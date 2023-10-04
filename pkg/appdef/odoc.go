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
	d := &oDoc{
		typ: makeType(app, name, TypeKind_ODoc),
	}
	d.fields = makeFields(d)
	d.containers = makeContainers(d)
	d.makeSysFields()
	app.appendType(d)
	return d
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
	r := &oRecord{
		typ: makeType(app, name, TypeKind_ORecord),
	}
	r.fields = makeFields(r)
	r.containers = makeContainers(r)
	r.makeSysFields()
	app.appendType(r)
	return r
}
