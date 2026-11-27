/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

/*
 * Copyright (c) 2026-present unTill Software Development Group B. V.
 */

package stateprovide

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
	"github.com/voedger/voedger/pkg/state/safestate"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/storages"
)

func wlogStateForTest(t *testing.T, events *coreutils.MockEvents) *hostState {
	t.Helper()
	s := newHostState(t.Context(), t.Name(), 0, nil)
	s.addStorage(sys.Storage_WLog, storages.NewWLogStorage(t.Context(),
		func() istructs.IEvents { return events }, state.SimpleWSIDFunc(1)), S_GET|S_READ)
	return s
}

func expectWLogRead(t *testing.T, require *require.Assertions, events *coreutils.MockEvents, offset istructs.Offset, count int, event *coreutils.MockPLogEvent) {
	t.Helper()
	events.On("ReadWLog", t.Context(), istructs.WSID(1), offset, count, mock.Anything).
		Run(func(args mock.Arguments) {
			require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(offset, event))
		}).Return(nil).Once()
}

func TestWLogBatchReleasesOnlyUndeliveredEventsOnCallbackError(t *testing.T) {
	require := require.New(t)
	for _, name := range []string{"host", "bundled host"} {
		t.Run(name, func(t *testing.T) {
			events := &coreutils.MockEvents{}
			host := wlogStateForTest(t, events)
			var s istructs.IState = host
			if name == "bundled host" {
				s = &bundledHostState{hostState: host}
			}
			first, second := &coreutils.MockPLogEvent{}, &coreutils.MockPLogEvent{}
			keys := make([]istructs.IStateKeyBuilder, 0, 2)
			for i, event := range []*coreutils.MockPLogEvent{first, second} {
				event.On("Release").Once()
				offset := i + 1
				expectWLogRead(t, require, events, istructs.Offset(offset), 1, event)
				key, err := s.KeyBuilder(sys.Storage_WLog, appdef.NullQName)
				require.NoError(err)
				key.PutInt64(sys.Storage_WLog_Field_Offset, int64(offset))
				keys = append(keys, key)
			}
			var retained istructs.IWLogEvent
			err := s.MustExistAll(keys, func(_ istructs.IKeyBuilder, value istructs.IStateValue, ok bool) error {
				require.True(ok)
				retained = value.(istructs.IStateWLogValue).AsEvent()
				return errTest
			})
			require.ErrorIs(err, errTest)
			require.Same(first, retained)
			first.AssertNotCalled(t, "Release")
			second.AssertExpectations(t)
			retained.Release()
			first.AssertExpectations(t)
		})
	}
}

func TestWLogExistenceCheckReleasesTheFoundEvent(t *testing.T) {
	require := require.New(t)
	events := &coreutils.MockEvents{}
	s := wlogStateForTest(t, events)
	event := &coreutils.MockPLogEvent{}
	event.On("Release").Once()
	expectWLogRead(t, require, events, 1, 1, event)
	key, err := s.KeyBuilder(sys.Storage_WLog, appdef.NullQName)
	require.NoError(err)
	key.PutInt64(sys.Storage_WLog_Field_Offset, 1)
	require.ErrorIs(s.MustNotExistAll([]istructs.IStateKeyBuilder{key}), ErrExists)
	event.AssertExpectations(t)
}

func TestWLogExtensionHandlesReleaseOnCallbackAndInvocationExit(t *testing.T) {
	require := require.New(t)
	events := &coreutils.MockEvents{}
	api := safestate.Provide(wlogStateForTest(t, events), nil)
	t.Cleanup(api.Release)
	row, lookup := &coreutils.MockPLogEvent{}, &coreutils.MockPLogEvent{}
	row.On("Release").Once()
	lookup.On("Release").Once()
	cud := &coreutils.MockCUDRow{}
	cud.On("AsInt32", "amount").Return(int32(42))
	row.On("CUDs", mock.Anything).Run(func(args mock.Arguments) {
		args.Get(0).(func(istructs.ICUDRow) bool)(cud)
	}).Once()
	expectWLogRead(t, require, events, 1, istructs.ReadToTheEnd, row)
	expectWLogRead(t, require, events, 2, 1, lookup)

	rangeKey := api.KeyBuilder(sys.Storage_WLog.String(), "")
	api.KeyBuilderPutInt64(rangeKey, sys.Storage_WLog_Field_Offset, 1)
	api.KeyBuilderPutInt64(rangeKey, sys.Storage_WLog_Field_Count, int64(istructs.ReadToTheEnd))
	lookupKey := api.KeyBuilder(sys.Storage_WLog.String(), "")
	api.KeyBuilderPutInt64(lookupKey, sys.Storage_WLog_Field_Offset, 2)
	var root, child, single safe.TValue
	require.PanicsWithValue(errTest, func() {
		api.ReadValues(rangeKey, func(_ safe.TKey, value safe.TValue) {
			root = value
			child = api.ValueGetAsValue(api.ValueAsValue(root, sys.Storage_WLog_Field_CUDs), 0)
			require.Equal(int32(42), api.ValueAsInt32(child, "amount"))
			single = api.MustGetValue(lookupKey)
			panic(errTest)
		})
	})
	row.AssertExpectations(t)
	require.PanicsWithValue(safestate.PanicIncorrectValue, func() { api.ValueAsInt64(root, sys.Storage_WLog_Field_Offset) })
	require.PanicsWithValue(safestate.PanicIncorrectValue, func() { api.ValueAsInt32(child, "amount") })
	lookup.AssertNotCalled(t, "Release")
	require.Equal(int64(2), api.ValueAsInt64(single, sys.Storage_WLog_Field_Offset))
	api.Release()
	lookup.AssertExpectations(t)
	require.PanicsWithValue(safestate.PanicIncorrectValue, func() { api.ValueAsInt64(single, sys.Storage_WLog_Field_Offset) })
}

func TestNestedWLogReadExpiresOnlyItsOwnHandles(t *testing.T) {
	require := require.New(t)
	events := &coreutils.MockEvents{}
	api := safestate.Provide(wlogStateForTest(t, events), nil)
	t.Cleanup(api.Release)

	key := api.KeyBuilder(sys.Storage_WLog.String(), "")
	api.KeyBuilderPutInt64(key, sys.Storage_WLog_Field_Offset, 1)
	api.KeyBuilderPutInt64(key, sys.Storage_WLog_Field_Count, 1)
	outer, inner := &coreutils.MockPLogEvent{}, &coreutils.MockPLogEvent{}
	for _, event := range []*coreutils.MockPLogEvent{outer, inner} {
		event.On("CUDs", mock.Anything).Once()
		event.On("Release").Once()
		expectWLogRead(t, require, events, 1, 1, event)
	}

	var outerCUDs safe.TValue
	api.ReadValues(key, func(_ safe.TKey, outerRow safe.TValue) {
		var innerCUDs safe.TValue
		api.ReadValues(key, func(_ safe.TKey, innerRow safe.TValue) {
			innerCUDs = api.ValueAsValue(innerRow, sys.Storage_WLog_Field_CUDs)
			outerCUDs = api.ValueAsValue(outerRow, sys.Storage_WLog_Field_CUDs)
			require.Zero(api.ValueLen(innerCUDs))
		})
		inner.AssertExpectations(t)
		outer.AssertNotCalled(t, "Release")
		require.PanicsWithValue(safestate.PanicIncorrectValue, func() { api.ValueLen(innerCUDs) })
		require.Zero(api.ValueLen(outerCUDs))
	})
	outer.AssertExpectations(t)
	require.PanicsWithValue(safestate.PanicIncorrectValue, func() { api.ValueLen(outerCUDs) })
}
