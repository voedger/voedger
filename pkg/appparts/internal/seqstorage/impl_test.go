/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	testWSQName := appdef.NewQName("test", "ws")
	testCDocQName := appdef.NewQName("test", "cdoc")
	mockEvents := &coreutils.MockEvents{}
	appDefBuilder := builder.New()
	ws := appDefBuilder.AddWorkspace(testWSQName)
	ws.AddCDoc(testCDocQName).AddField("IntFld", appdef.DataKind_int32, false)
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	appStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqSysVVMStorage := storage.NewSeqStorage(appStorage)
	seqStorage := New(istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)

	err = seqStorage.WriteValues([]isequencer.SeqValue{
		{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 1},
		{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 2},
		{Key: isequencer.NumberKey{WSID: 2, SeqID: 1}, Value: 3},
		{Key: isequencer.NumberKey{WSID: 2, SeqID: 2}, Value: 1},
	})
	require.NoError(err)

	err = seqStorage.WriteNextPLogOffset(isequencer.PLogOffset(5))
	require.NoError(err)

	numbers, err := seqStorage.ReadNumbers(1, []isequencer.SeqID{1})
	require.NoError(err)
	require.Equal()


	// t.Run("read empty", func(t *testing.T) {
	// 	t.Run("ReadNumbers", func(t *testing.T) {
	// 		numbers, err := seqStorage.ReadNumbers(1, nil)
	// 		require.NoError(err)
	// 		require.Empty(numbers)
	// 	})

	// 	t.Run("ReadPLogOffset", func(t *testing.T) {
	// 		offset, err := seqStorage.ReadNextPLogOffset()
	// 		require.NoError(err)
	// 		require.Equal(isequencer.PLogOffset(0), offset)
	// 	})

	// 	t.Run("Actualize", func(t *testing.T) {
	// 		ctx := context.Background()
	// 		err := seqStorage.ActualizeSequencesFromPLog(ctx, isequencer.PLogOffset(1), func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error {
	// 			t.Fail()
	// 			return nil
	// 		})
	// 		require.NoError(err)
	// 	})
	// })
}
