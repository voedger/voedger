/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package structures

import "github.com/voedger/voedger/pkg/appdef"

// # Supports:
//   - appdef.IGDoc
type GDoc struct {
	Doc
}

func NewGDoc(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName) *GDoc {
	return &GDoc{
		Doc: MakeDoc(app, ws, name, appdef.TypeKind_GDoc),
	}
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

func NewGRecord(app appdef.IAppDef, ws appdef.IWorkspace, name appdef.QName) *GRecord {
	return &GRecord{
		ContainedRecord: MakeContainedRecord(app, ws, name, appdef.TypeKind_GRecord),
	}
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
