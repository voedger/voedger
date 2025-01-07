/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IODoc
type ODoc struct {
	Doc
}

func NewODoc(ws appdef.IWorkspace, name appdef.QName) *ODoc {
	o := &ODoc{Doc: MakeDoc(ws, name, appdef.TypeKind_ODoc)}
	types.Propagate(o)
	return o
}

func (ODoc) IsODoc() {}

// # Supports:
//   - appdef.IODocBuilder
type ODocBuilder struct {
	DocBuilder
	*ODoc
}

func NewODocBuilder(d *ODoc) *ODocBuilder {
	return &ODocBuilder{
		DocBuilder: MakeDocBuilder(&d.Doc),
		ODoc:       d,
	}
}

// # Supports:
//   - appdef.IORecord
type ORecord struct {
	ContainedRecord
}

func (ORecord) IsORecord() {}

func NewORecord(ws appdef.IWorkspace, name appdef.QName) *ORecord {
	r := &ORecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_ORecord)}
	types.Propagate(r)
	return r
}

// # Supports:
//   - appdef.IORecordBuilder
type ORecordBuilder struct {
	ContainedRecordBuilder
	*ORecord
}

func NewORecordBuilder(r *ORecord) *ORecordBuilder {
	return &ORecordBuilder{
		ContainedRecordBuilder: MakeContainedRecordBuilder(&r.ContainedRecord),
		ORecord:                r,
	}
}
