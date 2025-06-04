/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sys

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// sys workspace descriptor
var SysWSKind = appdef.NewQName(appdef.SysPackage, "sysWS")

// Projection offsets view
type ProjectionOffsetViewFields struct {
	Partition string
	Projector string
	Offset    string
}

type ProjectionOffsetView struct {
	Name   appdef.QName
	Fields ProjectionOffsetViewFields
}

var ProjectionOffsetsView = ProjectionOffsetView{
	Name: appdef.NewQName(appdef.SysPackage, "projectionOffsets"),
	Fields: ProjectionOffsetViewFields{
		Partition: "partition",
		Projector: "projector",
		Offset:    "offset",
	},
}

// Child workspaces IDs view
type NextBaseWSIDViewFields struct {
	PartKeyDummy  string
	ClustColDummy string
	NextBaseWSID  string
}

var NextBaseWSIDView = struct {
	Name   appdef.QName
	Fields NextBaseWSIDViewFields
}{
	Name: appdef.NewQName(appdef.SysPackage, "NextBaseWSID"),
	Fields: NextBaseWSIDViewFields{
		PartKeyDummy:  "dummy1",
		ClustColDummy: "dummy2",
		NextBaseWSID:  "NextBaseWSID",
	},
}

// Records registry view
type RecordsRegistryViewFields struct {
	IDHi       string
	ID         string
	WLogOffset string
	QName      string
	IsActive   string
}

func (RecordsRegistryViewFields) CrackID(id istructs.RecordID) int64 {
	return int64(id >> RecordsRegistryView.partBits) // nolint G115
}

var RecordsRegistryView = struct {
	Name     appdef.QName
	Fields   RecordsRegistryViewFields
	partBits int
}{
	Name: appdef.NewQName(appdef.SysPackage, "RecordsRegistry"),
	Fields: RecordsRegistryViewFields{
		IDHi:       "IDHi",
		ID:         "ID",
		WLogOffset: "WLogOffset",
		QName:      "QName",
		IsActive:   "IsActive",
	},
	partBits: 18, // 18 high bits for IDHi
}
