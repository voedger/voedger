/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"context"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestQueryProcessorState(t *testing.T) {

	require := require.New(t)
	sentObjects := make([]istructs.IObject, 0)

	execQueryCallbackFunc := func() istructs.ExecQueryCallback {
		return func(object istructs.IObject) error {
			sentObjects = append(sentObjects, object)
			return nil
		}
	}

	qps := ProvideQueryProcessorStateFactory()(context.Background(), nil, nil, nil, nil, nil, nil, nil, nil, nil, istructs.NewNullObjectBuilder, nil, execQueryCallbackFunc, state.StateConfig{})
	kb, err := qps.KeyBuilder(sys.Storage_Result, appdef.NullQName)
	require.NoError(err)
	require.NotNil(kb)
	rows := queryProcessorStateMaxIntents + 1
	for i := 0; i < rows; i++ {
		vb, err := qps.NewValue(kb)
		require.NoError(err)
		require.NotNil(vb)
	}

	intent := qps.FindIntent(kb)
	require.NotNil(intent)

	err = qps.ApplyIntents()
	require.NoError(err)
	require.Len(sentObjects, rows)
	require.NotEqual(unsafe.Pointer(&sentObjects[0]), unsafe.Pointer(&sentObjects[1]))
}
