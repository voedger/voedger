/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

func applyDeployApp(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	kb, err := st.KeyBuilder(state.View, QNameViewDeployedApps)
	if err != nil {
		// notest
		return err
	}
	appName := event.ArgumentObject().AsString(Field_Name)
	appQName, err := istructs.ParseAppQName(appName)
	if err != nil {
		// parsed already by c.cluster.DeployApp
		// notest
		return err
	}
	clusterAppID := istructs.ClusterApps[appQName] // parsed already by c.cluster.DeplyApp
	kb.PutInt32(Field_ClusterAppID, int32(clusterAppID))
	kb.PutString(Field_Name, appName)
	sv, err := intents.NewValue(kb)
	if err != nil {
		// notest
		return err
	}
	sv.PutInt64(Field_DeployEventWLogOffset, int64(event.WLogOffset()))
	return nil
}
