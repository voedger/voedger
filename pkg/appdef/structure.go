/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Structure
//
// # Implements:
//	 - IStructure
//	 - IStructureBuilder
type structure struct {
	typ
	fields
	containers
	uniques
	withAbstract
}

// Makes new structure
func makeStructure(app *appDef, name QName, kind TypeKind, parent interface{}) structure {
	s := structure{
		typ: makeType(app, name, kind),
	}
	s.fields = makeFields(parent)
	s.fields.makeSysFields(kind)
	s.containers = makeContainers(parent)
	s.uniques = makeUniques(parent)
	return s
}

func (s structure) SystemField_QName() IField {
	return s.fields.Field(SystemField_QName)
}

// # Record
//
// # Implements:
//	- IRecord
//	- IRecordBuilder
type record struct {
	structure
}

func (r record) SystemField_ID() IField {
	return r.fields.Field(SystemField_ID)
}

// Makes new record
func makeRecord(app *appDef, name QName, kind TypeKind, parent interface{}) record {
	r := record{
		structure: makeStructure(app, name, kind, parent),
	}
	return r
}

// # Document
//
// # Implements:
//	- IDoc
//	- IDocBuilder
type doc struct {
	record
}

func (d doc) isDoc() {}

// Makes new document
func makeDoc(app *appDef, name QName, kind TypeKind, parent interface{}) doc {
	d := doc{
		record: makeRecord(app, name, kind, parent),
	}
	return d
}

// # Contained record
//
// # Implements:
//	- IContainedRecord
//	- IContainedRecordBuilder
type containedRecord struct {
	record
}

// Makes new record
func makeContainedRecord(app *appDef, name QName, kind TypeKind, parent interface{}) containedRecord {
	r := containedRecord{
		record: makeRecord(app, name, kind, parent),
	}
	return r
}

func (r containedRecord) SystemField_ParentID() IField {
	return r.fields.Field(SystemField_ParentID)
}

func (r containedRecord) SystemField_Container() IField {
	return r.fields.Field(SystemField_Container)
}
