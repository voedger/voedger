/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import (
	"errors"

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
	fields.WithFields
	containers.WithContainers
	uniques.WithUniques
	abstracts.WithAbstract
}

// Makes new structure
func MakeStructure(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Structure {
	s := Structure{
		Typ:            types.MakeType(ws.App(), ws, name, kind),
		WithFields:     fields.MakeWithFields(ws, kind),
		WithContainers: containers.MakeWithContainers(ws, kind),
		WithAbstract:   abstracts.MakeWithAbstract(),
	}
	s.WithFields.MakeSysFields()
	s.WithUniques = uniques.MakeWithUniques(ws.App().Type, &s.WithFields)
	return s
}

func (s Structure) SystemField_QName() appdef.IField {
	return s.WithFields.Field(appdef.SystemField_QName)
}

func (s *Structure) Validate() error {
	return errors.Join(
		fields.ValidateTypeFields(s),
		containers.ValidateTypeContainers(s),
	)
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
		FieldsBuilder:       fields.MakeFieldsBuilder(&structure.WithFields),
		ContainersBuilder:   containers.MakeContainersBuilder(&structure.WithContainers),
		UniquesBuilder:      uniques.MakeUniquesBuilder(&structure.WithUniques),
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
	return r.WithFields.Field(appdef.SystemField_ID)
}

// Makes new record
func MakeRecord(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Record {
	r := Record{
		Structure: MakeStructure(ws, name, kind),
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
func MakeDoc(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) Doc {
	return Doc{Record: MakeRecord(ws, name, kind)}
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
func MakeContainedRecord(ws appdef.IWorkspace, name appdef.QName, kind appdef.TypeKind) ContainedRecord {
	return ContainedRecord{Record: MakeRecord(ws, name, kind)}
}

func (r ContainedRecord) SystemField_ParentID() appdef.IField {
	return r.WithFields.Field(appdef.SystemField_ParentID)
}

func (r ContainedRecord) SystemField_Container() appdef.IField {
	return r.WithFields.Field(appdef.SystemField_Container)
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
