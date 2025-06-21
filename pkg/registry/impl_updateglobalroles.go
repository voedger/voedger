/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package registry

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideUpdateGlobalRoles(cfgRegistry *istructsmem.AppConfigType) {
	cfgRegistry.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandUpdateGlobalRoles,
		cmdCommandUpdateGlobalRolesExec,
	))
}

// sys/registry/pseudoWSID
// auth: System
// [~server.authnz.groles/cmp.c.sys.UpdateGlobalRoles~impl]
func cmdCommandUpdateGlobalRolesExec(args istructs.ExecCommandArgs) (err error) {
	login := args.ArgumentObject.AsString(field_Login)
	appName := args.ArgumentObject.AsString(field_AppName)
	globalRoles := args.ArgumentObject.AsString(field_GlobalRoles)
	return UpdateGlobalRoles(login, args.State, args.Intents, args.WSID, appName, globalRoles)
}
