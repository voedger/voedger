/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestWLogStorage_Read(t *testing.T) {
	require := require.New(t)
	t.Run("Each callback receives its own event", func(t *testing.T) {
		rows := []*coreutils.MockPLogEvent{{}, {}}
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", t.Context(), istructs.WSID(1), istructs.FirstOffset, len(rows), mock.Anything).
			Run(func(args mock.Arguments) {
				for i, row := range rows {
					row.On("Release").Once()
					require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(istructs.FirstOffset+istructs.Offset(i), row))
					row.AssertExpectations(t)
				}
			}).Return(nil)
		storage := NewWLogStorage(t.Context(), func() istructs.IEvents { return events }, state.SimpleWSIDFunc(1))
		kb := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		kb.PutInt64(sys.Storage_WLog_Field_Count, int64(len(rows)))
		calls := 0
		require.NoError(storage.(state.IWithRead).Read(kb, func(key istructs.IKey, value istructs.IStateValue) error {
			event := value.(istructs.IStateWLogValue).AsEvent()
			defer event.Release()
			require.Same(rows[calls], event)
			require.Equal(int64(calls+1), key.AsInt64(sys.Storage_WLog_Field_Offset))
			calls++
			return nil
		}))
		require.Equal(len(rows), calls)
	})
	t.Run("Read returns the WLog error", func(t *testing.T) {
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", t.Context(), istructs.WSID(1), istructs.FirstOffset, 1, mock.Anything).Return(errTest)
		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(t.Context(), eventsFunc, state.SimpleWSIDFunc(1))
		withRead := storage.(state.IWithRead)
		k := storage.NewKeyBuilder(appdef.NullQName, nil)
		k.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		k.PutInt64(sys.Storage_WLog_Field_Count, 1)
		err := withRead.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })
		require.ErrorIs(err, errTest)
	})
}

func TestWLogStorage_Get(t *testing.T) {
	require := require.New(t)
	get := func(t *testing.T, event istructs.IWLogEvent, readErr error) (istructs.IStateValue, error) {
		t.Helper()
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", t.Context(), istructs.WSID(1), istructs.FirstOffset, 1, mock.Anything).
			Run(func(args mock.Arguments) {
				cb := args.Get(4).(istructs.WLogEventsReaderCallback)
				require.NoError(cb(istructs.FirstOffset, event))
			}).Return(readErr).Once()
		storage := NewWLogStorage(t.Context(), func() istructs.IEvents { return events }, state.SimpleWSIDFunc(1))
		key := storage.NewKeyBuilder(appdef.NullQName, nil)
		key.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		return storage.(state.IWithGet).Get(key)
	}

	t.Run("The caller owns the returned event", func(t *testing.T) {
		event := &coreutils.MockPLogEvent{}
		event.On("Release").Once()
		t.Cleanup(func() { event.AssertExpectations(t) })
		sv, err := get(t, event, nil)
		require.NoError(err)
		require.NotNil(sv)
		actual := sv.(istructs.IStateWLogValue).AsEvent()
		defer actual.Release()
		require.Same(event, actual)
		event.AssertNotCalled(t, "Release")
	})

	t.Run("Nested CUD values are readable while the caller holds the event", func(t *testing.T) {
		event := storedEventFixture(t, require, appdef.NewQName("main", "Command"), 1)
		sv, err := get(t, event, nil)
		require.NoError(err)
		require.NotNil(sv)
		defer sv.(istructs.IStateWLogValue).AsEvent().Release()
		require.Equal(int64(1), sv.AsInt64(sys.Storage_WLog_Field_Offset))

		cuds := sv.AsValue(sys.Storage_WLog_Field_CUDs)
		require.Equal(1, cuds.Length())
		cud := cuds.GetAsValue(0)

		require.Equal(int32(1), cud.AsInt32("x"))
		require.PanicsWithValue(errFieldByIndexIsNotAnObjectOrArray, func() { sv.GetAsValue(0) })
		require.PanicsWithError(errValueFieldUndefined(sys.Storage_WLog_Field_CUDs).Error(), func() { cud.AsValue(sys.Storage_WLog_Field_CUDs) })
		require.PanicsWithValue(errCurrentValueIsNotAnArray, func() { sv.GetAsInt64(0) })
		require.PanicsWithError(errRecordIDFieldUndefined(sys.Storage_WLog_Field_Offset).Error(), func() { sv.AsRecordID(sys.Storage_WLog_Field_Offset) })
	})
	t.Run("Get releases the undelivered event if the read fails after the callback", func(t *testing.T) {
		event := &coreutils.MockPLogEvent{}
		event.On("Release").Once()
		value, err := get(t, event, errTest)
		require.ErrorIs(err, errTest)
		require.Nil(value)
		event.AssertExpectations(t)
	})
}

func TestWLogKeyBuilder(t *testing.T) {
	require := require.New(t)
	t.Run("String", func(t *testing.T) {
		s := &wLogStorage{
			wsidFunc: func() istructs.WSID { return istructs.WSID(42) },
		}
		kb := s.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Count, 10)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 20)
		kb.PutInt64(sys.Storage_WLog_Field_WSID, 30)

		require.Equal("storage:sys.WLog, wsid:30, offset:20, count:10", kb.(fmt.Stringer).String())
	})
}
