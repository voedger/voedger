/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/uniques"
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

func provideExecQueryApp(asp istructs.IAppStructsProvider) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		appQNameStr := args.ArgumentObject.AsString(Field_AppQName)
		appQName, err := istructs.ParseAppQName(appQNameStr)
		if err != nil {
			return fmt.Errorf("failed to parse appQName %q: %w", appQNameStr, err)
		}

		as, err := asp.AppStructs(appQName)
		if err != nil {
			// notest
			return err
		}
		appID, err := uniques.GetRecordIDByUniqueCombination(args.WSID, qNameWDocApp, as, map[string]interface{}{
			Field_AppQName: appQNameStr,
		})
		if err != nil {
			return err
		}
		if appID == istructs.NullRecordID {
			return nil
		}

		kb, err := args.State.KeyBuilder(state.Record, qNameWDocApp)
		if err != nil {
			// notest
			return err
		}
		kb.PutRecordID(state.Field_ID, appID)
		appRec, err := args.State.MustExist(kb)
		if err != nil {
			// notest
			return err
		}

		return callback(&res{
			numPartitions:    appRec.AsInt32(Field_NumPartitions),
			numAppWorkspaces: appRec.AsInt32(Field_NumAppWorkspaces),
		})
	}
}
