/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
)

func provideExecDeployApp(asp istructs.IAppStructsProvider) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		appQNameStr := args.ArgumentObject.AsString(Field_AppQName)
		appQName, err := istructs.ParseAppQName(appQNameStr)
		if err != nil {
			// notest
			return err
		}
		kb, err := args.State.KeyBuilder(state.Record, qNameWDocApp)
		if err != nil {
			// notest
			return err
		}
		vb, err := args.Intents.NewValue(kb)
		if err != nil {
			// notest
			return err
		}
		vb.PutRecordID(state.Field_ID, 1)
		vb.PutString(Field_AppQName, appQNameStr)
		vb.PutInt32(Field_NumPartitions, args.ArgumentObject.AsInt32(Field_NumPartitions))
		vb.PutInt32(Field_NumAppWorkspaces, args.ArgumentObject.AsInt32(Field_NumAppWorkspaces))

		// deploy app workspaces
		pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
		wLogOffset := istructs.FirstOffset
		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			return err
		}
		for wsNum := 0; istructs.NumAppWorkspaces(wsNum) < as.NumAppWorkspaces(); wsNum++ {
			appwsutils.InitAppWS(as)
		}
		return nil
	}
}
