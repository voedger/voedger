/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"context"
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

type mockCUDRow struct {
	istructs.ICUDRow
	mock.Mock
}

type mockWLogEvent struct {
	mock.Mock
}

func (e *mockWLogEvent) ArgumentObject() istructs.IObject {
	return e.Called().Get(0).(istructs.IObject)
}
func (e *mockWLogEvent) Bytes() []byte { return e.Called().Get(0).([]byte) }
func (e *mockWLogEvent) CUDs(cb func(rec istructs.ICUDRow) bool) {
	e.Called(cb)
}
func (e *mockWLogEvent) RegisteredAt() istructs.UnixMilli {
	return e.Called().Get(0).(istructs.UnixMilli)
}
func (e *mockWLogEvent) DeviceID() istructs.ConnectedDeviceID {
	return e.Called().Get(0).(istructs.ConnectedDeviceID)
}
func (e *mockWLogEvent) Synced() bool                 { return e.Called().Bool(0) }
func (e *mockWLogEvent) QName() appdef.QName          { return e.Called().Get(0).(appdef.QName) }
func (e *mockWLogEvent) SyncedAt() istructs.UnixMilli { return e.Called().Get(0).(istructs.UnixMilli) }
func (e *mockWLogEvent) Error() istructs.IEventError  { return e.Called().Get(0).(istructs.IEventError) }
func (e *mockWLogEvent) Release()                     {}

func TestWLogStorage_Read(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		touched := false
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(istructs.FirstOffset, nil))
			})

		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(context.Background(), eventsFunc, state.SimpleWSIDFunc(istructs.WSID(1)))
		withRead := storage.(state.IWithRead)
		kb := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		kb.PutInt64(sys.Storage_WLog_Field_Count, 1)
		require.NoError(withRead.Read(kb, func(key istructs.IKey, _ istructs.IStateValue) (err error) {
			touched = true
			require.Equal(int64(1), key.AsInt64(sys.Storage_WLog_Field_Offset))
			return err
		}))

		require.True(touched)
	})
	t.Run("Should return error on read wlog", func(t *testing.T) {
		require := require.New(t)
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).Return(errTest)
		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(context.Background(), eventsFunc, state.SimpleWSIDFunc(istructs.WSID(1)))
		withRead := storage.(state.IWithRead)
		k := storage.NewKeyBuilder(appdef.NullQName, nil)
		k.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		k.PutInt64(sys.Storage_WLog_Field_Count, 1)
		err := withRead.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })
		require.ErrorIs(err, errTest)
	})
}

func TestWLogStorage_Get(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		event := new(mockWLogEvent)
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				cb := args.Get(4).(istructs.WLogEventsReaderCallback)
				require.NoError(cb(istructs.FirstOffset, event))
			})
		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(context.Background(), eventsFunc, state.SimpleWSIDFunc(istructs.WSID(1)))
		withGet := storage.(state.IWithGet)

		kb := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		sv, err := withGet.Get(kb)
		require.NoError(err)
		require.NotNil(sv)
	})

}
func TestWLogStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		event := new(mockWLogEvent)
		event.On("CUDs", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(rec istructs.ICUDRow) bool)
			cb(new(mockCUDRow))
		})
		events := &coreutils.MockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				cb := args.Get(4).(istructs.WLogEventsReaderCallback)
				require.NoError(cb(istructs.FirstOffset, event))
			})
		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(context.Background(), eventsFunc, state.SimpleWSIDFunc(istructs.WSID(1)))
		withGet := storage.(state.IWithGet)
		kb := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		kb.PutInt64(sys.Storage_WLog_Field_Count, 1)

		sv, err := withGet.Get(kb)
		require.NoError(err)
		require.NotNil(sv)
		require.Equal(int64(1), sv.AsInt64(sys.Storage_WLog_Field_Offset))

		cuds := sv.AsValue(sys.Storage_WLog_Field_CUDs)
		cud := cuds.GetAsValue(0)

		require.Equal(1, cuds.Length())
		require.NotNil(cud)
		require.PanicsWithValue(errFieldByIndexIsNotAnObjectOrArray, func() { sv.GetAsValue(0) })
		require.PanicsWithError(errValueFieldUndefined(sys.Storage_WLog_Field_CUDs).Error(), func() { cud.AsValue(sys.Storage_WLog_Field_CUDs) })
		require.PanicsWithValue(errCurrentValueIsNotAnArray, func() { sv.GetAsInt64(0) })
		require.PanicsWithError(errRecordIDFieldUndefined(sys.Storage_WLog_Field_Offset).Error(), func() { sv.AsRecordID(sys.Storage_WLog_Field_Offset) })
	})
	t.Run("Should return error when error occurred on read wlog", func(t *testing.T) {
		require := require.New(t)
		events := &coreutils.MockEvents{}
		events.
			On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(istructs.FirstOffset, nil))
			}).
			On("ReadWLog", context.Background(), istructs.WSID(1), istructs.Offset(2), 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(errTest)
		eventsFunc := func() istructs.IEvents { return events }
		storage := NewWLogStorage(context.Background(), eventsFunc, state.SimpleWSIDFunc(istructs.WSID(1)))
		withGet := storage.(state.IWithGet)

		kb1 := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb1.PutInt64(sys.Storage_WLog_Field_Offset, 1)
		kb1.PutInt64(sys.Storage_WLog_Field_Count, 1)
		kb2 := storage.NewKeyBuilder(appdef.NullQName, nil)
		kb2.PutInt64(sys.Storage_WLog_Field_Offset, 2)
		kb2.PutInt64(sys.Storage_WLog_Field_Count, 1)

		v1, err1 := withGet.Get(kb1)
		require.NoError(err1)
		require.NotNil(v1)

		v2, err2 := withGet.Get(kb2)
		require.Nil(v2)
		require.ErrorIs(err2, errTest)
	})
}

func TestWLogKeyBuilder(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		s := &wLogStorage{
			wsidFunc: func() istructs.WSID { return istructs.WSID(42) },
		}
		kb := s.NewKeyBuilder(appdef.NullQName, nil)
		kb.PutInt64(sys.Storage_WLog_Field_Count, 10)
		kb.PutInt64(sys.Storage_WLog_Field_Offset, 20)
		kb.PutInt64(sys.Storage_WLog_Field_WSID, 30)

		require.Equal(t, "storage:sys.WLog, wsid:30, offset:20, count:10", kb.(fmt.Stringer).String())
	})
}
