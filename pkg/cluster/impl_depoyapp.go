/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func execDeployApp(args istructs.ExecCommandArgs) (err error) {
	appName := args.ArgumentObject.AsString(Field_Name)
	appQName, err := istructs.ParseAppQName(appName)
	if err != nil {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("failed to parse app qname %q: %s", appName, err.Error()))
	}
	clusterAppID, ok := istructs.ClusterApps[appQName]
	if !ok {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("cluster app id is unknown for app %s", appName))
	}

	// kb, err := args.State.KeyBuilder(state.View, QNameViewDeployedApps)
	// if err != nil {
	// 	// notest
	// 	return err
	// }
	// kb.PutInt32(Field_ClusterAppID, int32(clusterAppID))
	// kb.PutString(Field_Name, appName)
	// _, deployed, err := args.State.CanExist(kb)
	// if err == nil {
	// 	// notest
	// 	return err
	// }
	// if deployed {
	// 	return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf("app %s is deployed already", appName))
	// }

	// deploy app workspace
	

	return nil
}
