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
//   - appdef.ICDoc
type CDoc struct {
	SingletonDoc
}

// Creates a new CDoc
func NewCDoc(ws appdef.IWorkspace, name appdef.QName) *CDoc {
	d := &CDoc{SingletonDoc: MakeSingleton(ws, name, appdef.TypeKind_CDoc)}
	types.Propagate(d)
	return d
}

func (CDoc) IsCDoc() {}

// # Supports:
//   - appdef.ICDocBuilder
type CDocBuilder struct {
	SingletonBuilder
	d *CDoc
}

func NewCDocBuilder(d *CDoc) *CDocBuilder {
	return &CDocBuilder{
		SingletonBuilder: MakeSingletonBuilder(&d.SingletonDoc),
		d:                d,
	}
}

// # Supports:
//   - appdef.ICRecord
type CRecord struct {
	ContainedRecord
}

func NewCRecord(ws appdef.IWorkspace, name appdef.QName) *CRecord {
	r := &CRecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_CRecord)}
	types.Propagate(r)
	return r
}

func (CRecord) IsCRecord() {}

// # Supports:
//   - appdef.ICRecordBuilder
type CRecordBuilder struct {
	ContainedRecordBuilder
	*CRecord
}

func NewCRecordBuilder(r *CRecord) *CRecordBuilder {
	return &CRecordBuilder{
		ContainedRecordBuilder: MakeContainedRecordBuilder(&r.ContainedRecord),
		CRecord:                r,
	}
}
