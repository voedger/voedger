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

	t.Run("Write and Read Offset", func(t *testing.T) {
		actualOffset, err := seqStorage.ReadNextPLogOffset()
		require.NoError(err)
		require.Zero(actualOffset)

		err = seqStorage.WriteNextPLogOffset(isequencer.PLogOffset(5))
		require.NoError(err)

		actualOffset, err = seqStorage.ReadNextPLogOffset()
		require.NoError(err)
		require.Equal(isequencer.PLogOffset(5), actualOffset)

		err = seqStorage.WriteNextPLogOffset(isequencer.PLogOffset(6))
		require.NoError(err)

		actualOffset, err = seqStorage.ReadNextPLogOffset()
		require.NoError(err)
		require.Equal(isequencer.PLogOffset(6), actualOffset)
	})

	// ReadNumbers
	t.Run("Write and Read Numbers", func(t *testing.T) {
		err = seqStorage.WriteValues([]isequencer.SeqValue{
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 1},
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 6},
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 3}, Value: 8},
			{Key: isequencer.NumberKey{WSID: 2, SeqID: 1}, Value: 3},
			{Key: isequencer.NumberKey{WSID: 2, SeqID: 2}, Value: 1},
		})
		require.NoError(err)

		cases := []struct {
			wsid            isequencer.WSID
			seqIDs          [][]isequencer.SeqID
			expectedNumbers [][]isequencer.Number
		}{
			{
				wsid: 1,
				seqIDs: [][]isequencer.SeqID{
					{},
					{1},
					{2},
					{3},
					{4},
					{1, 2},
					{1, 2, 3},
					{1, 2, 3, 4},
					{4, 3, 2, 1},
				},
				expectedNumbers: [][]isequencer.Number{
					{},
					{1},
					{6},
					{8},
					{0},
					{1, 6},
					{1, 6, 8},
					{1, 6, 8, 0},
					{0, 8, 6, 1},
				},
			},
			{
				wsid: 2,
				seqIDs: [][]isequencer.SeqID{
					{1},
					{2},
					{3},
				},
				expectedNumbers: [][]isequencer.Number{
					{3},
					{1},
					{0},
				},
			},
		}
		for _, c := range cases {
			for i, seqIDs := range c.seqIDs {
				numbers, err := seqStorage.ReadNumbers(c.wsid, seqIDs)
				require.NoError(err)
				require.Equal(c.expectedNumbers[i], numbers, seqIDs)
			}
		}
	})

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

func TestSeqIDMapping(t *testing.T) {
	require := require.New(t)
	mockEvents := &coreutils.MockEvents{}
	appDefBuilder := builder.New()
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	appStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqSysVVMStorage := storage.NewSeqStorage(appStorage)
	seqStorage := New(istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)
	require.Equal(istructs.QNameIDWLogOffsetSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameWLogOffsetSequence])
	require.Equal(istructs.QNameIDCRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameCRecordIDSequence])
	require.Equal(istructs.QNameIDCRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameCRecordIDSequence])
}
