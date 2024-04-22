/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

type res struct {
	istructs.IObject
	numPartitions    int32
	numAppWorkspaces int32
}

func (r *res) AsInt32(name string) int32 {
	if name == Field_NumPartitions {
		return r.numPartitions
	}
	return r.numAppWorkspaces
}

func execQueryApp(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	appQNameStr := args.ArgumentObject.AsString(Field_AppQName)

	appQName, err := istructs.ParseAppQName(appQNameStr)
	if err != nil {
		return fmt.Errorf("failed to parse appQName %q: %w", appQNameStr, err)
	}

	clusterAppID, ok := istructs.ClusterApps[appQName]
	if !ok {
		return fmt.Errorf("cluster app ID is unknown for the app %s", appQName)
	}

	kb, err := args.State.KeyBuilder(state.View, QNameViewDeployedApps)
	if err != nil {
		// notest
		return err
	}

	kb.PutInt32(Field_ClusterAppID, int32(clusterAppID))
	kb.PutString(Field_AppQName, appQName.String())
	v, ok, err := args.State.CanExist(kb)
	if err != nil {
		// notest
		return err
	}
	if !ok {
		return nil
	}

	if kb, err = args.State.KeyBuilder(state.WLog, appdef.NullQName); err != nil {
		// notest
		return err
	}
	kb.PutInt64(state.Field_Offset, v.AsInt64(Field_DeployEventWLogOffset))
	kb.PutInt64(state.Field_Count, 1)
	eventSV, err := args.State.MustExist(kb)
	if err != nil {
		// notest
		return err
	}

	event := eventSV.AsEvent("").(istructs.IWLogEvent)
	return callback(&res{
		numPartitions:    event.ArgumentObject().AsInt32(Field_NumPartitions),
		numAppWorkspaces: event.ArgumentObject().AsInt32(Field_NumAppWorkspaces),
	})
}
