/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package wskinds

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func ProvideCDocsWorkspaceKinds(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddStruct(authnz.QNameCDoc_WorkspaceKind_DeviceProfile, appdef.DefKind_CDoc).SetSingleton()
	appDefBuilder.AddStruct(authnz.QNameCDoc_WorkspaceKind_UserProfile, appdef.DefKind_CDoc).
		AddField(authnz.Field_DisplayName, appdef.DataKind_string, false). // made not required according to https://dev.untill.com/projects/#!613071
		SetSingleton()

	// See UserProfile.Email at packages/air/userprofile/provide.go

	appDefBuilder.AddStruct(authnz.QNameCDoc_WorkspaceKind_AppWorkspace, appdef.DefKind_CDoc).SetSingleton()
}
