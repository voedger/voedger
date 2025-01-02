/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
)

// # Supports:
//   - appdef.IGDoc
type GDoc struct {
	Doc
}

func (GDoc) IsGDoc() {}

func NewGDoc(ws appdef.IWorkspace, name appdef.QName) *GDoc {
	d := &GDoc{Doc: MakeDoc(ws, name, appdef.TypeKind_GDoc)}
	types.Propagate(d)
	return d
}

// # Supports:
//   - appdef.IGDocBuilder
type GDocBuilder struct {
	DocBuilder
	*GDoc
}

func NewGDocBuilder(d *GDoc) *GDocBuilder {
	return &GDocBuilder{
		DocBuilder: MakeDocBuilder(&d.Doc),
		GDoc:       d,
	}
}

// # Supports:
//   - appdef.IGRecord
type GRecord struct {
	ContainedRecord
}

func (GRecord) IsGRecord() {}

func NewGRecord(ws appdef.IWorkspace, name appdef.QName) *GRecord {
	r := &GRecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_GRecord)}
	types.Propagate(r)
	return r
}

// # Supports:
//   - appdef.IGRecordBuilder
type GRecordBuilder struct {
	ContainedRecordBuilder
	*GRecord
}

func NewGRecordBuilder(r *GRecord) *GRecordBuilder {
	return &GRecordBuilder{
		ContainedRecordBuilder: MakeContainedRecordBuilder(&r.ContainedRecord),
		GRecord:                r,
	}
}
