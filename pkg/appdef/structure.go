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
}

// Makes new structure
func makeStructure(app *appDef, name QName, kind TypeKind) structure {
	s := structure{
		typ: makeType(app, name, kind),
	}
	s.fields = makeFields(&s)
	s.containers = makeContainers(&s)
	s.uniques = makeUniques(&s)
	return s
}

// Document is a structure.
//
// # Implements:
//	- IDoc
type doc struct {
	structure
}

// Makes new document
func makeDoc(app *appDef, name QName, kind TypeKind) doc {
	d := doc{
		structure: makeStructure(app, name, kind),
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
func makeRecord(app *appDef, name QName, kind TypeKind) record {
	r := record{
		structure: makeStructure(app, name, kind),
	}
	return r
}
