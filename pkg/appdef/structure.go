/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Structure is a type with fields, containers and uniques.
//
// # Implements:
//	 - IStructure
type structure struct {
	typ
	fields
	containers
	uniques
	withAbstract
	parent interface{}
}

// Makes new structure
func makeStructure(app *appDef, name QName, kind TypeKind, parent interface{}) structure {
	s := structure{
		typ:    makeType(app, name, kind),
		parent: parent,
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

// Document is a structure.
//
// # Implements:
//	- IDoc
type doc struct {
	structure
}

func (d doc) SystemField_ID() IField {
	return d.fields.Field(SystemField_ID)
}

func (d doc) isDoc() {}

// Makes new document
func makeDoc(app *appDef, name QName, kind TypeKind, parent interface{}) doc {
	d := doc{
		structure: makeStructure(app, name, kind, parent),
	}
	return d
}

// Record is a structure.
//
// # Implements:
//	- IRecord
type record struct {
	structure
}

// Makes new record
func makeRecord(app *appDef, name QName, kind TypeKind, parent interface{}) record {
	r := record{
		structure: makeStructure(app, name, kind, parent),
	}
	return r
}

func (r record) SystemField_ID() IField {
	return r.fields.Field(SystemField_ID)
}

func (r record) SystemField_ParentID() IField {
	return r.fields.Field(SystemField_ParentID)
}

func (r record) SystemField_Container() IField {
	return r.fields.Field(SystemField_Container)
}
