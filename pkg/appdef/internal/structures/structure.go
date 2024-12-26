/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/abstracts"
	"github.com/voedger/voedger/pkg/appdef/internal/containers"
	"github.com/voedger/voedger/pkg/appdef/internal/fields"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/appdef/internal/uniques"
)

// # Supports:
//   - appdef.IStructure
type Structure struct {
	types.Typ
	fields.Fields
	containers.Containers
	uniques.Uniques
	abstracts.WithAbstract
}

// Makes new structure
func MakeStructure(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Structure {
	s := Structure{
		Typ:          types.MakeType(app, ws, name, kind),
		Fields:       fields.MakeFields(app, ws, kind),
		Containers:   containers.MakeContainers(app, kind),
		WithAbstract: abstracts.MakeWithAbstract(),
	}
	s.Fields.MakeSysFields()
	s.Uniques = uniques.MakeUniques(app, &s.Fields)
	return s
}

func (s Structure) SystemField_QName() appdef.IField {
	return s.Fields.Field(appdef.SystemField_QName)
}

// # Supports:
//   - appdef.IStructureBuilder
type StructureBuilder struct {
	types.TypeBuilder
	fields.FieldsBuilder
	containers.ContainersBuilder
	uniques.UniquesBuilder
	abstracts.WithAbstractBuilder
	*Structure
}

func MakeStructureBuilder(structure *Structure) StructureBuilder {
	return StructureBuilder{
		TypeBuilder:         types.MakeTypeBuilder(&structure.Typ),
		FieldsBuilder:       fields.MakeFieldsBuilder(&structure.Fields),
		ContainersBuilder:   containers.MakeContainersBuilder(&structure.Containers),
		UniquesBuilder:      uniques.MakeUniquesBuilder(&structure.Uniques),
		WithAbstractBuilder: abstracts.MakeWithAbstractBuilder(&structure.WithAbstract),
		Structure:           structure,
	}
}

// # Supports:
//   - appdef.IRecord
type Record struct {
	Structure
}

func (r Record) SystemField_ID() appdef.IField {
	return r.Fields.Field(appdef.SystemField_ID)
}

// Makes new record
func MakeRecord(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Record {
	r := Record{
		Structure: MakeStructure(app, ws, name, kind),
	}
	return r
}

// # Supports:
//   - appdef.IRecordBuilder
type RecordBuilder struct {
	StructureBuilder
	*Record
}

func MakeRecordBuilder(record *Record) RecordBuilder {
	return RecordBuilder{
		StructureBuilder: MakeStructureBuilder(&record.Structure),
		Record:           record,
	}
}

// # Supports:
//   - appdef.IDoc
type Doc struct {
	Record
}

// Makes new document
func MakeDoc(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Doc {
	d := Doc{
		Record: MakeRecord(app, ws, name, kind),
	}
	return d
}

// # Supports:
//   - appdef.IDocBuilder
type DocBuilder struct {
	RecordBuilder
	*Doc
}

func MakeDocBuilder(doc *Doc) DocBuilder {
	return DocBuilder{
		RecordBuilder: MakeRecordBuilder(&doc.Record),
		Doc:           doc,
	}
}

// # Supports:
//   - appdef.IContainedRecord
type ContainedRecord struct {
	Record
}

// Makes new record
func MakeContainedRecord(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) ContainedRecord {
	return ContainedRecord{Record: MakeRecord(app, ws, name, kind)}
}

func (r ContainedRecord) SystemField_ParentID() appdef.IField {
	return r.Fields.Field(appdef.SystemField_ParentID)
}

func (r ContainedRecord) SystemField_Container() appdef.IField {
	return r.Fields.Field(appdef.SystemField_Container)
}

// # Supports:
//   - appdef.IContainedRecordBuilder
type ContainedRecordBuilder struct {
	RecordBuilder
	*ContainedRecord
}

func MakeContainedRecordBuilder(record *ContainedRecord) ContainedRecordBuilder {
	return ContainedRecordBuilder{
		RecordBuilder:   MakeRecordBuilder(&record.Record),
		ContainedRecord: record,
	}
}
