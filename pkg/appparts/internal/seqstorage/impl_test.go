/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"context"
	"strconv"
	"testing"

	gomock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vit/mock"
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
	seqSysVVMStorage := storage.NewVVMSeqStorageAdapter(appStorage)
	seqStorage := New(istructs.ClusterApps[istructs.AppQName_test1_app1], istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)

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

type expectedSeqValue struct {
	wsid   uint64
	seqID  uint16
	number uint64
}

type obj struct {
	qName      appdef.QName
	id         uint64 // >0 -> will be passed to NewRecordID, <0 -> abs(id) will be used. Need to test "old" ids according to issue 688
	containers []obj
}

type testPLogEvent struct {
	qName         appdef.QName
	offset        istructs.Offset
	wsid          uint64
	cuds          []obj
	expectedBatch []expectedSeqValue
}

func TestActualizeFromPLog(t *testing.T) {
	require := require.New(t)
	testWSQName := appdef.NewQName("test", "ws")
	testCDocQName := appdef.NewQName("test", "cdoc")
	testCRecordQName := appdef.NewQName("test", "crecord")
	testORecordQName := appdef.NewQName("test", "orecord")
	testWRecordQName := appdef.NewQName("test", "wrecord")
	testWDocQName := appdef.NewQName("test", "wdoc")
	testODocQName := appdef.NewQName("test", "odoc")
	testCmdQName := appdef.NewQName("test", "cmd")

	appDefBuilder := builder.New()
	ws := appDefBuilder.AddWorkspace(testWSQName)
	ws.AddCRecord(testCRecordQName)
	ws.AddORecord(testORecordQName)
	ws.AddWRecord(testWRecordQName)
	ws.AddCDoc(testCDocQName).AddContainer("crecord", testCRecordQName, appdef.Occurs_Unbounded, appdef.Occurs_Unbounded)
	ws.AddODoc(testODocQName).AddContainer("orecord", testORecordQName, appdef.Occurs_Unbounded, appdef.Occurs_Unbounded)
	ws.AddWDoc(testWDocQName).AddContainer("wrecord", testWRecordQName, appdef.Occurs_Unbounded, appdef.Occurs_Unbounded)
	ws.AddCommand(testCmdQName)
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	appStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqSysVVMStorage := storage.NewVVMSeqStorageAdapter(appStorage)

	plogs := [][]testPLogEvent{
		{
			{qName: testCmdQName, wsid: 1, offset: 1, cuds: []obj{{qName: testCDocQName, id: 1}}, expectedBatch: []expectedSeqValue{
				{wsid: 1, seqID: istructs.QNameIDCRecordIDSequence, number: 1},
			}},
		},
		{
			{qName: testCmdQName, wsid: 1, offset: 1, cuds: []obj{{qName: testCDocQName, id: 1}}, expectedBatch: []expectedSeqValue{
				{wsid: 1, seqID: istructs.QNameIDCRecordIDSequence, number: 1},
			}},
			{qName: testCmdQName, wsid: 2, offset: 1, cuds: []obj{{qName: testCDocQName, id: 2}}, expectedBatch: []expectedSeqValue{
				{wsid: 2, seqID: istructs.QNameIDCRecordIDSequence, number: 2},
			}},
			{qName: testCmdQName, wsid: 3, offset: 1, cuds: []obj{{qName: testCDocQName, id: 3}}, expectedBatch: []expectedSeqValue{
				{wsid: 3, seqID: istructs.QNameIDCRecordIDSequence, number: 3},
			}},
		},
		{
			{qName: testCmdQName, wsid: 1, offset: 1, cuds: []obj{
				{qName: testCDocQName, id: 1, containers: []obj{
					{qName: testCRecordQName, id: 2},
					{qName: testCRecordQName, id: 3},
				}},
				{qName: testWDocQName, id: 4, containers: []obj{
					{qName: testWRecordQName, id: 5},
				},
			}}, expectedBatch: []expectedSeqValue{
				{wsid: 1, seqID: istructs.QNameIDCRecordIDSequence, number: 4},
				{wsid: 1, seqID: istructs.QNameIDOWRecordIDSequence, number: 5},
			}},
		},
	}

	for i, plogEvents := range plogs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mockEvents := &coreutils.MockEvents{}
			mockEvents.On("ReadPLog", gomock.Anything, gomock.Anything, gomock.Anything, gomock.Anything, gomock.Anything).
				Return(nil).
				Run(func(args gomock.Arguments) {
					cb := args.Get(4).(istructs.PLogEventsReaderCallback)
					for _, pLogEvent := range plogEvents {
						iPLogEvent := testPLogEventToIPlogEvent(pLogEvent)
						require.NoError(cb(1, iPLogEvent))
					}
				})
			seqStorage := New(istructs.ClusterApps[istructs.AppQName_test1_app1], istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)
			numEvent := 0
			seqStorage.ActualizeSequencesFromPLog(context.Background(), 1, func(batch []isequencer.SeqValue, offset isequencer.PLogOffset) error {
				require.Equal(isequencer.PLogOffset(plogEvents[numEvent].offset), offset)
				expectedBatch := []isequencer.SeqValue{}
				for _, eb := range plogEvents[numEvent].expectedBatch {
					expectedBatch = append(expectedBatch, isequencer.SeqValue{
						Key:   isequencer.NumberKey{WSID: isequencer.WSID(eb.wsid), SeqID: isequencer.SeqID(eb.seqID)},
						Value: isequencer.Number(istructs.NewCDocCRecordID(istructs.RecordID(eb.number))),
					})
				}
				require.Equal(expectedBatch, batch)
				numEvent++
				return nil
			})
			require.Len(plogEvents, numEvent)
		})
	}

}

func testPLogEventToIPlogEvent(pLogEvent testPLogEvent) istructs.IPLogEvent {
	mockEvent := coreutils.MockPLogEvent{}
	mockEvent.On("CUDs", mock.Anything).Run(func(args gomock.Arguments) {
		cudCallback := args[0].(func(istructs.ICUDRow) bool)
		for _, cudTemplate := range pLogEvent.cuds {
			cud := coreutils.TestObject{
				Name:   cudTemplate.qName,
				Id:     istructs.NewCDocCRecordID(istructs.RecordID(cudTemplate.id)),
				IsNew_: true,
			}
			if !cudCallback(&cud) {
				panic("")
			}
		}
	})
	argObj := coreutils.TestObject{
		Name: pLogEvent.qName,
	}
	mockEvent.On("Workspace").Return(istructs.WSID(pLogEvent.wsid))
	mockEvent.On("ArgumentObject").Return(&argObj)
	return &mockEvent
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
	seqSysVVMStorage := storage.NewVVMSeqStorageAdapter(appStorage)
	seqStorage := New(istructs.ClusterApps[istructs.AppQName_test1_app1], istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)
	require.Equal(istructs.QNameIDWLogOffsetSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameWLogOffsetSequence])
	require.Equal(istructs.QNameIDCRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameCRecordIDSequence])
	require.Equal(istructs.QNameIDCRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameCRecordIDSequence])
}
