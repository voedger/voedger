/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package wskinds

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func ProvideCDocsWorkspaceKinds(appDefBuilder appdef.IAppDefBuilder) {
	appDefBuilder.AddCDoc(authnz.QNameCDoc_WorkspaceKind_DeviceProfile).SetSingleton()

	cDoc := appDefBuilder.AddCDoc(authnz.QNameCDoc_WorkspaceKind_UserProfile)
	cDoc.AddField(authnz.Field_DisplayName, appdef.DataKind_string, false) // made not required according to https://dev.untill.com/projects/#!613071
	cDoc.SetSingleton()

	// See UserProfile.Email at packages/air/userprofile/provide.go

	appDefBuilder.AddCDoc(authnz.QNameCDoc_WorkspaceKind_AppWorkspace).SetSingleton()
}
