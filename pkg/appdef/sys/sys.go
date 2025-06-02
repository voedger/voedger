/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package sys

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/datas"
)

func MakeSysPackage(adb appdef.IAppDefBuilder) {
	adb.AddPackage(appdef.SysPackage, appdef.SysPackagePath)

	makeSysWorkspace(adb)
}

func makeSysWorkspace(adb appdef.IAppDefBuilder) {
	wsb := adb.AddWorkspace(appdef.SysWorkspaceQName)

	// make sys data types
	ws := wsb.Workspace()
	for k := appdef.DataKind_null + 1; k < appdef.DataKind_FakeLast; k++ {
		_ = datas.NewSysData(ws, k)
	}

	// workspace descriptor
	_ = wsb.AddCDoc(SysWSKind)
	wsb.SetDescriptor(SysWSKind)

	// for projectors: sys.projectionOffsets
	viewProjectionOffsets := wsb.AddView(ProjectionOffsetsView.Name)
	viewProjectionOffsets.Key().PartKey().AddField(ProjectionOffsetsView.Fields.Partition, appdef.DataKind_int32)
	viewProjectionOffsets.Key().ClustCols().AddField(ProjectionOffsetsView.Fields.Projector, appdef.DataKind_QName)
	viewProjectionOffsets.Value().AddField(ProjectionOffsetsView.Fields.Offset, appdef.DataKind_int64, true)

	// for child workspaces: sys.NextBaseWSID
	viewNextBaseWSID := wsb.AddView(NextBaseWSIDView.Name)
	viewNextBaseWSID.Key().PartKey().AddField(NextBaseWSIDView.Fields.PartKeyDummy, appdef.DataKind_int32)
	viewNextBaseWSID.Key().ClustCols().AddField(NextBaseWSIDView.Fields.ClustColDummy, appdef.DataKind_int32)
	viewNextBaseWSID.Value().AddField(NextBaseWSIDView.Fields.NextBaseWSID, appdef.DataKind_int64, true)
}
