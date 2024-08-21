/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestQueryContextStorage(t *testing.T) {
	wsid := istructs.NullWSID
	arg := istructs.NewNullObject()
	wsidFunc := state.SimpleWSIDFunc(wsid)
	argFunc := func() istructs.IObject { return arg }
	storage := NewQueryContextStorage(argFunc, wsidFunc)
	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	v, err := storage.(state.IWithGet).Get(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(sys.Storage_QueryContext_Field_Workspace))
	require.NotNil(t, v.AsValue(sys.Storage_QueryContext_Field_ArgumentObject))
}
