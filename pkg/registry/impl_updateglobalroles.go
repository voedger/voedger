/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package registry

import (
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideUpdateGlobalRoles(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(
		QNameCommandUpdateGlobalRoles,
		cmdCommandUpdateGlobalRolesExec,
	))
}

// sys/registry/pseudoWSID
// auth: System
// [~server.authnz.groles/cmp.c.registry.UpdateGlobalRoles~impl]
func cmdCommandUpdateGlobalRolesExec(args istructs.ExecCommandArgs) error {
	login := args.ArgumentObject.AsString(field_Login)
	appName := args.ArgumentObject.AsString(field_AppName)
	globalRoles := args.ArgumentObject.AsString(field_GlobalRoles)
	if len(globalRoles) > 0 {
		globalRolesStr := strings.Split(globalRoles, ",")
		for _, role := range globalRolesStr {
			_, err := appdef.ParseQName(role)
			if err != nil {
				return coreutils.NewHTTPErrorf(http.StatusBadRequest, err)
			}
		}
	}
	return UpdateGlobalRoles(login, args.State, args.Intents, args.WSID, appName, globalRoles)
}
