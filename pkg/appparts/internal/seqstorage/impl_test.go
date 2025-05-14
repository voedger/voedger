/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package seqstorage

import (
	"context"
	"testing"

	gomock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vit/mock"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

var (
	testWSQName      = appdef.NewQName("test", "ws")
	testCDocQName    = appdef.NewQName("test", "cdoc")
	testCRecordQName = appdef.NewQName("test", "crecord")
	testORecordQName = appdef.NewQName("test", "orecord")
	testWRecordQName = appdef.NewQName("test", "wrecord")
	testWDocQName    = appdef.NewQName("test", "wdoc")
	testODocQName    = appdef.NewQName("test", "odoc")
	testCmdQName     = appdef.NewQName("test", "cmd")
)

func TestReadWrite(t *testing.T) {
	require := require.New(t)
	testWSQName := appdef.NewQName("test", "ws")
	testCDocQName := appdef.NewQName("test", "cdoc")
	mockEvents := &coreutils.MockEvents{}
	appDefBuilder := builder.New()
	ws := appDefBuilder.AddWorkspace(testWSQName)
	ws.AddCDoc(testCDocQName).AddField("IntFld", appdef.DataKind_int32, false)
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	seqStorage := setupSeqStorage(t, mockEvents, appDef)

	// read from empty storage
	actualOffset, err := seqStorage.ReadNextPLogOffset()
	require.NoError(err)
	require.Zero(actualOffset)

	numbers, err := seqStorage.ReadNumbers(1, []isequencer.SeqID{})
	require.NoError(err)
	require.Empty(numbers)
	numbers, err = seqStorage.ReadNumbers(1, []isequencer.SeqID{1})
	require.NoError(err)
	require.Zero(numbers[0])

	require.NoError(err)

	// will overwrite sequences and offset 5 times with new value
	for counter := range 5 {
		numberBump := isequencer.Number(counter)
		expectedPLogOffset := isequencer.PLogOffset(42 + counter)
		err = seqStorage.WriteValuesAndNextPLogOffset([]isequencer.SeqValue{
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 1}, Value: 1 + numberBump},
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 2}, Value: 6 + numberBump},
			{Key: isequencer.NumberKey{WSID: 1, SeqID: 3}, Value: 8 + numberBump},
			{Key: isequencer.NumberKey{WSID: 2, SeqID: 1}, Value: 3 + numberBump},
			{Key: isequencer.NumberKey{WSID: 2, SeqID: 2}, Value: 1 + numberBump},
		}, expectedPLogOffset)
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
			for i, seqIDsTemplate := range c.seqIDs {
				numbers, err := seqStorage.ReadNumbers(c.wsid, seqIDsTemplate)
				require.NoError(err)
				expectedNumbers := []isequencer.Number{}
				for _, expectedNumber := range c.expectedNumbers[i] {
					if expectedNumber == 0 {
						expectedNumbers = append(expectedNumbers, expectedNumber)
					} else {
						expectedNumbers = append(expectedNumbers, expectedNumber+isequencer.Number(counter))
					}
				}
				require.Equal(expectedNumbers, numbers, seqIDsTemplate)

				actualPLogOffset, err := seqStorage.ReadNextPLogOffset()
				require.NoError(err)
				require.Equal(expectedPLogOffset, actualPLogOffset)
			}
		}
	}
}

func TestSequenceActualization(t *testing.T) {
	require := require.New(t)
	appDef := setupTestAppDef(t)

	// Test case definitions
	cases := []struct {
		name string
		plog []testPLogEvent
	}{
		{
			name: "one event with no cuds",
			plog: []testPLogEvent{{qName: testCmdQName, wsid: 1, pLogOffset: 1, wLogOffset: 2,
				expectedBatch: []expectedSeqValue{
					{wsid: 1, seqID: istructs.QNameIDWLogOffsetSequence, number: 2},
				}}},
		},
		{
			name: "one event with one cud",
			plog: []testPLogEvent{{qName: testCmdQName, wsid: 1, pLogOffset: 1, wLogOffset: 2, cuds: []cud{{qName: testCDocQName, id: 1}},
				expectedBatch: []expectedSeqValue{
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 1},
					{wsid: 1, seqID: istructs.QNameIDWLogOffsetSequence, number: 2},
				}}},
		},
		{
			name: "3 events, 2nd has 2 cuds, other - 1 cud",
			plog: []testPLogEvent{
				// 1st event
				{qName: testCmdQName, wsid: 1, pLogOffset: 1, wLogOffset: 2, cuds: []cud{{qName: testCDocQName, id: 1}},
					expectedBatch: []expectedSeqValue{
						{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 1},
						{wsid: 1, seqID: istructs.QNameIDWLogOffsetSequence, number: 2},
					}},
				// 2nd event
				{qName: testCmdQName, wsid: 2, pLogOffset: 2, wLogOffset: 3, cuds: []cud{
					{qName: testCDocQName, id: 2},
					{qName: testWDocQName, id: 3},
				}, expectedBatch: []expectedSeqValue{
					{wsid: 2, seqID: istructs.QNameIDRecordIDSequence, number: 2},
					{wsid: 2, seqID: istructs.QNameIDRecordIDSequence, number: 3},
					{wsid: 2, seqID: istructs.QNameIDWLogOffsetSequence, number: 3},
				}},
				// 3rd event
				{qName: testCmdQName, wsid: 3, pLogOffset: 3, wLogOffset: 4, cuds: []cud{{qName: testCDocQName, id: 3}},
					expectedBatch: []expectedSeqValue{
						{wsid: 3, seqID: istructs.QNameIDRecordIDSequence, number: 3},
						{wsid: 3, seqID: istructs.QNameIDWLogOffsetSequence, number: 4},
					}},
			},
		},
		{
			name: "1 event with few cdocs, wdocs and records",
			plog: []testPLogEvent{
				{qName: testCmdQName, wsid: 1, pLogOffset: 1, wLogOffset: 2, cuds: []cud{
					{qName: testCDocQName, id: 1},
					{qName: testCRecordQName, id: 2},
					{qName: testCRecordQName, id: 3},
					{qName: testWDocQName, id: 4},
					{qName: testWRecordQName, id: 5},
				}, expectedBatch: []expectedSeqValue{
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 1},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 2},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 3},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 4},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 5},
					{wsid: 1, seqID: istructs.QNameIDWLogOffsetSequence, number: 2},
				}},
			},
		},
		{
			name: "arg: odoc with 2 orecords + 2 new cuds + 1 update cud (should be skipped)",
			plog: []testPLogEvent{
				{qName: testODocQName, wsid: 1, pLogOffset: 1, wLogOffset: 2, arg: obj{
					cud: cud{qName: testODocQName, id: 1},
					containers: []obj{
						{cud: cud{qName: testORecordQName, id: 2}},
						{cud: cud{qName: testORecordQName, id: 3}},
					},
				}, cuds: []cud{
					{qName: testCDocQName, id: 4},
					{qName: testCDocQName, id: 123456789, isOld: true},
				}, expectedBatch: []expectedSeqValue{
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 1},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 2},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 3},
					{wsid: 1, seqID: istructs.QNameIDRecordIDSequence, number: 4},
					{wsid: 1, seqID: istructs.QNameIDWLogOffsetSequence, number: 2},
				}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockEvents := &coreutils.MockEvents{}
			mockEvents.On("ReadPLog", gomock.Anything, gomock.Anything, gomock.Anything, gomock.Anything, gomock.Anything).
				Return(nil).
				Run(func(args gomock.Arguments) {
					cb := args.Get(4).(istructs.PLogEventsReaderCallback)
					for _, pLogEvent := range tc.plog {
						iPLogEvent := testPLogEventToIPlogEvent(pLogEvent, appDef)
						require.NoError(cb(pLogEvent.pLogOffset, iPLogEvent))
					}
				})

			seqStorage := setupSeqStorage(t, mockEvents, appDef)

			processingCount := 0
			err := seqStorage.ActualizeSequencesFromPLog(context.Background(), 1, func(ctx context.Context, batch []isequencer.SeqValue, offset isequencer.PLogOffset) error {
				require.Equal(isequencer.PLogOffset(tc.plog[processingCount].pLogOffset), offset, "Offset mismatch in event %d", processingCount)

				expectedBatch := buildExpectedBatch(tc.plog[processingCount].expectedBatch)
				require.Equal(expectedBatch, batch, "Batch mismatch in event %d", processingCount)

				processingCount++
				return nil
			})

			require.NoError(err)
			require.Equal(len(tc.plog), processingCount, "Not all events were processed")
		})
	}
}

func TestSeqIDMapping(t *testing.T) {
	require := require.New(t)
	mockEvents := &coreutils.MockEvents{}
	appDefBuilder := builder.New()
	appDef, err := appDefBuilder.Build()
	require.NoError(err)
	appStorageProvider := provider.Provide(mem.Provide(testingu.MockTime))
	appStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqSysVVMStorage := storage.NewVVMSeqStorageAdapter(appStorage)
	seqStorage := New(istructs.ClusterApps[istructs.AppQName_test1_app1], istructs.PartitionID(1), mockEvents, appDef, seqSysVVMStorage)
	require.Equal(istructs.QNameIDWLogOffsetSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameWLogOffsetSequence])
	require.Equal(istructs.QNameIDRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameRecordIDSequence])
	require.Equal(istructs.QNameIDRecordIDSequence, seqStorage.(*implISeqStorage).seqIDs[istructs.QNameRecordIDSequence])
}

// buildExpectedBatch converts expectedSeqValue entries to isequencer.SeqValue batch
func buildExpectedBatch(expectedValues []expectedSeqValue) []isequencer.SeqValue {
	expectedBatch := []isequencer.SeqValue{}
	for _, ev := range expectedValues {
		id := istructs.RecordID(ev.number)
		expectedBatch = append(expectedBatch, isequencer.SeqValue{
			Key:   isequencer.NumberKey{WSID: isequencer.WSID(ev.wsid), SeqID: isequencer.SeqID(ev.seqID)},
			Value: isequencer.Number(id),
		})
	}
	return expectedBatch
}

func testPLogEventToIPlogEvent(pLogEvent testPLogEvent, appDef appdef.IAppDef) istructs.IPLogEvent {
	mockEvent := coreutils.MockPLogEvent{}
	mockEvent.On("CUDs", mock.Anything).Run(func(args gomock.Arguments) {
		cudCallback := args[0].(func(istructs.ICUDRow) bool)
		for _, cudTemplate := range pLogEvent.cuds {
			cud := coreutils.TestObject{
				Name:   cudTemplate.qName,
				ID_:    cudTemplate.ID(appDef),
				IsNew_: !cudTemplate.isOld,
			}
			if !cudCallback(&cud) {
				panic("")
			}
		}
	})
	argObj := coreutils.TestObject{
		Name:        pLogEvent.qName,
		Containers_: map[string][]*coreutils.TestObject{},
		ID_:         pLogEvent.arg.ID(appDef),
	}
	argKind := appDef.Type(pLogEvent.qName).Kind()
	if argKind == appdef.TypeKind_ODoc {
		// handle case when odoc is the arg
		for _, oDocContainer := range pLogEvent.arg.containers {
			argObj.Containers_[oDocContainer.qName.Entity()] = append(argObj.Containers_[oDocContainer.qName.Entity()], &coreutils.TestObject{
				Name:   oDocContainer.qName,
				ID_:    oDocContainer.ID(appDef),
				IsNew_: !oDocContainer.isOld,
			})
		}
	}
	mockEvent.On("Workspace").Return(istructs.WSID(pLogEvent.wsid))
	mockEvent.On("ArgumentObject").Return(&argObj)
	mockEvent.On("WLogOffset").Return(pLogEvent.wLogOffset)
	return &mockEvent
}

// setupTestAppDef creates a standard app definition for testing
func setupTestAppDef(t *testing.T) appdef.IAppDef {
	require := require.New(t)
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
	return appDef
}

// setupSeqStorage creates and returns a sequence storage instance for testing
func setupSeqStorage(t *testing.T, mockEvents *coreutils.MockEvents, appDef appdef.IAppDef) isequencer.ISeqStorage {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(testingu.MockTime))
	appStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqSysVVMStorage := storage.NewVVMSeqStorageAdapter(appStorage)
	return New(istructs.ClusterApps[istructs.AppQName_test1_app1], istructs.PartitionID(42), mockEvents, appDef, seqSysVVMStorage)
}

type expectedSeqValue struct {
	wsid   uint64
	seqID  uint16
	number uint64
}

type cud struct {
	qName   appdef.QName
	id      uint64
	exactID istructs.RecordID
	isOld   bool // !IsNew
}

func (c cud) ID(appDef appdef.IAppDef) istructs.RecordID {
	if c.exactID != istructs.NullRecordID {
		return c.exactID
	}
	return istructs.RecordID(c.id)
}

type obj struct {
	cud
	containers []obj
}

type testPLogEvent struct {
	qName         appdef.QName
	pLogOffset    istructs.Offset
	wLogOffset    istructs.Offset
	wsid          uint64
	cuds          []cud
	arg           obj
	expectedBatch []expectedSeqValue
}
