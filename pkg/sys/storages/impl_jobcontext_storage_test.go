/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
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

func TestJobContextStorage(t *testing.T) {
	wsid := istructs.NullWSID
	wsidFunc := state.SimpleWSIDFunc(wsid)
	unixTimeFunc := func() int64 { return 1234567890 }
	storage := NewJobContextStorage(wsidFunc, unixTimeFunc)
	kb := storage.NewKeyBuilder(appdef.NullQName, nil)
	require.Equal(t, sys.Storage_JobContext, kb.Storage())
	v, err := storage.(state.IWithGet).Get(kb)
	require.NoError(t, err)
	require.Equal(t, int64(wsid), v.AsInt64(sys.Storage_JobContext_Field_Workspace))
	require.Equal(t, int64(1234567890), v.AsInt64(sys.Storage_JobContext_Field_UnixTime))
}
