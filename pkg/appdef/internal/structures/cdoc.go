/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.ICDoc
type CDoc struct {
	SingletonDoc
}

// Creates a new CDoc
func NewCDoc(ws appdef.IWorkspace, name appdef.QName) *CDoc {
	return &CDoc{SingletonDoc: MakeSingleton(ws, name, appdef.TypeKind_CDoc)}
}

// # Supports:
//   - appdef.ICDocBuilder
type CDocBuilder struct {
	SingletonBuilder
	*CDoc
}

func NewCDocBuilder(cDoc *CDoc) *CDocBuilder {
	return &CDocBuilder{
		SingletonBuilder: MakeSingletonBuilder(&cDoc.SingletonDoc),
		CDoc:             cDoc,
	}
}

// # Supports:
//   - appdef.ICRecord
type CRecord struct {
	ContainedRecord
}

func NewCRecord(ws appdef.IWorkspace, name appdef.QName) *CRecord {
	return &CRecord{ContainedRecord: MakeContainedRecord(ws, name, appdef.TypeKind_CRecord)}
}

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
