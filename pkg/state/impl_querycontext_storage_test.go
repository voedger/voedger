/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestQueryContextStorage(t *testing.T) {
	wsid := istructs.NullWSID
	arg := istructs.NewNullObject()

	wsidFunc := SimpleWSIDFunc(wsid)

	execQueryArgsFunc := func() istructs.PrepareArgs {
		return istructs.PrepareArgs{
			Workpiece:      nil,
			ArgumentObject: arg,
			WSID:           wsid,
			Workspace:      nil,
		}
	}
	argFunc := func() istructs.IObject { return arg }

	s := ProvideQueryProcessorStateFactory()(context.Background(), nil, nil, wsidFunc, nil, nil, nil, nil, execQueryArgsFunc, argFunc, nil, nil, nil)

	kb, err := s.KeyBuilder(sys.Storage_QueryContext, appdef.NullQName)
	require.NoError(t, err)

	v, err := s.MustExist(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(sys.Storage_QueryContext_Field_Workspace))
	require.NotNil(t, v.AsValue(sys.Storage_QueryContext_Field_ArgumentObject))
}
