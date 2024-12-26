/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.ICDoc
type CDoc struct {
	Singleton
}

// Creates a new CDoc
func NewCDoc(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName) *CDoc {
	d := &CDoc{
		Singleton: MakeSingleton(app, ws, name, appdef.TypeKind_CDoc),
	}
	return d
}

// # Supports:
//   - appdef.ICDocBuilder
type CDocBuilder struct {
	SingletonBuilder
	*CDoc
}

func NewCDocBuilder(cDoc *CDoc) *CDocBuilder {
	return &CDocBuilder{
		SingletonBuilder: MakeSingletonBuilder(&cDoc.Singleton),
		CDoc:             cDoc,
	}
}

// # Supports:
//   - ICRecord
type CRecord struct {
	ContainedRecord
}

func NewCRecord(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName) *CRecord {
	r := &CRecord{
		ContainedRecord: MakeContainedRecord(app, ws, name, appdef.TypeKind_CRecord),
	}
	return r
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
