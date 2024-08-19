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

func TestCommandContextStorage(t *testing.T) {
	wsid := istructs.NullWSID
	arg := istructs.NewNullObject()
	unloggedArg := istructs.NewNullObject()

	wsidFunc := state.SimpleWSIDFunc(wsid)
	argFunc := func() istructs.IObject { return arg }
	unloggedArgFunc := func() istructs.IObject { return unloggedArg }
	wlogOffsetFunc := func() istructs.Offset { return 42 }

	storage := NewCommandContextStorage(argFunc, unloggedArgFunc, wsidFunc, wlogOffsetFunc)

	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	v, err := storage.(state.IWithGet).Get(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(sys.Storage_CommandContext_Field_Workspace))
	require.NotNil(t, v.AsValue(sys.Storage_CommandContext_Field_ArgumentObject))
	require.NotNil(t, v.AsValue(sys.Storage_CommandContext_Field_ArgumentUnloggedObject))
	require.Equal(t, int64(42), v.AsInt64(sys.Storage_CommandContext_Field_WLogOffset))
}
