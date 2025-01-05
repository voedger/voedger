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
//   - appdef.IWDoc
type WDoc struct {
	SingletonDoc
}

func (WDoc) IsWDoc() {}

func NewWDoc(ws appdef.IWorkspace, name appdef.QName) *WDoc {
	d := &WDoc{SingletonDoc: MakeSingleton(ws, name, appdef.TypeKind_WDoc)}
	types.Propagate(d)
	return d
}

// # Supports:
//   - appdef.IWDocBuilder
type WDocBuilder struct {
	SingletonBuilder
	*WDoc
}

func NewWDocBuilder(d *WDoc) *WDocBuilder {
	return &WDocBuilder{
		SingletonBuilder: MakeSingletonBuilder(&d.SingletonDoc),
		WDoc:             d,
	}
}

// # Supports:
//   - appdef.IWRecord
type WRecord struct {
	ContainedRecord
}

func (WRecord) IsWRecord() {}

func NewWRecord(ws appdef.IWorkspace, name appdef.QName) *WRecord {
	r := &WRecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_WRecord)}
	types.Propagate(r)
	return r
}

// # Supports:
//   - appdef.IWRecordBuilder
type WRecordBuilder struct {
	ContainedRecordBuilder
	*WRecord
}

func NewWRecordBuilder(r *WRecord) *WRecordBuilder {
	return &WRecordBuilder{
		ContainedRecordBuilder: MakeContainedRecordBuilder(&r.ContainedRecord),
		WRecord:                r,
	}
}
