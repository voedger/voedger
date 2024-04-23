/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//	 - IStructure
type structure struct {
	typ
	fields
	containers
	uniques
	withAbstract
}

// Makes new structure
func makeStructure(app *appDef, name QName, kind TypeKind) structure {
	s := structure{
		typ:          makeType(app, name, kind),
		fields:       makeFields(app, kind),
		containers:   makeContainers(app, kind),
		withAbstract: makeWithAbstract(),
	}
	s.fields.makeSysFields()
	s.uniques = makeUniques(app, &s.fields)
	return s
}

func (s structure) SystemField_QName() IField {
	return s.fields.Field(SystemField_QName)
}

// # Implements:
//	- IStructureBuilder
type structureBuilder struct {
	typeBuilder
	fieldsBuilder
	containersBuilder
	uniquesBuilder
	withAbstractBuilder
	*structure
}

func makeStructureBuilder(structure *structure) structureBuilder {
	return structureBuilder{
		typeBuilder:         makeTypeBuilder(&structure.typ),
		fieldsBuilder:       makeFieldsBuilder(&structure.fields),
		containersBuilder:   makeContainersBuilder(&structure.containers),
		uniquesBuilder:      makeUniquesBuilder(&structure.uniques),
		withAbstractBuilder: makeWithAbstractBuilder(&structure.withAbstract),
		structure:           structure,
	}
}

// # Implements:
//	- IRecord
type record struct {
	structure
}

func (r record) SystemField_ID() IField {
	return r.fields.Field(SystemField_ID)
}

// Makes new record
func makeRecord(app *appDef, name QName, kind TypeKind) record {
	r := record{
		structure: makeStructure(app, name, kind),
	}
	return r
}

// # Implements:
//	- IRecordBuilder
type recordBuilder struct {
	structureBuilder
	*record
}

func makeRecordBuilder(record *record) recordBuilder {
	return recordBuilder{
		structureBuilder: makeStructureBuilder(&record.structure),
		record:           record,
	}
}

// # Implements:
//	- IDoc
type doc struct {
	record
}

func (d doc) isDoc() {}

// Makes new document
func makeDoc(app *appDef, name QName, kind TypeKind) doc {
	d := doc{
		record: makeRecord(app, name, kind),
	}
	return d
}

// # Implements:
//	- IDocBuilder
type docBuilder struct {
	recordBuilder
	*doc
}

func makeDocBuilder(doc *doc) docBuilder {
	return docBuilder{
		recordBuilder: makeRecordBuilder(&doc.record),
		doc:           doc,
	}
}

// # Implements:
//	- IContainedRecord
type containedRecord struct {
	record
}

// Makes new record
func makeContainedRecord(app *appDef, name QName, kind TypeKind) containedRecord {
	r := containedRecord{
		record: makeRecord(app, name, kind),
	}
	return r
}

func (r containedRecord) SystemField_ParentID() IField {
	return r.fields.Field(SystemField_ParentID)
}

func (r containedRecord) SystemField_Container() IField {
	return r.fields.Field(SystemField_Container)
}

// # Implements:
//	- IContainedRecordBuilder
type containedRecordBuilder struct {
	recordBuilder
	*containedRecord
}

func makeContainedRecordBuilder(record *containedRecord) containedRecordBuilder {
	return containedRecordBuilder{
		recordBuilder:   makeRecordBuilder(&record.record),
		containedRecord: record,
	}
}
