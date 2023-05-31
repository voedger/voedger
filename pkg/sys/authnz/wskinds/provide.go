/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package wskinds

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func ProvideCDocsWorkspaceKinds(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddSingleton(authnz.QNameCDoc_WorkspaceKind_DeviceProfile)

	appDefBuilder.AddSingleton(authnz.QNameCDoc_WorkspaceKind_UserProfile).
		AddField(authnz.Field_DisplayName, appdef.DataKind_string, false) // made not required according to https://dev.untill.com/projects/#!613071

	// See UserProfile.Email at packages/air/userprofile/provide.go

	appDefBuilder.AddSingleton(authnz.QNameCDoc_WorkspaceKind_AppWorkspace)
}
