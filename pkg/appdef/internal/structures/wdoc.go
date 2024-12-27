/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.IWDoc
type WDoc struct {
	Singleton
}

func NewWDoc(ws appdef.IWorkspace, name appdef.QName) *WDoc {
	return &WDoc{Singleton: MakeSingleton(ws, name, appdef.TypeKind_WDoc)}
}

// # Supports:
//   - appdef.IWDocBuilder
type WDocBuilder struct {
	SingletonBuilder
	*WDoc
}

func NewWDocBuilder(d *WDoc) *WDocBuilder {
	return &WDocBuilder{
		SingletonBuilder: MakeSingletonBuilder(&d.Singleton),
		WDoc:             d,
	}
}

// # Supports:
//   - appdef.IWRecord
type WRecord struct {
	ContainedRecord
}

func NewWRecord(ws appdef.IWorkspace, name appdef.QName) *WRecord {
	return &WRecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_WRecord)}
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
