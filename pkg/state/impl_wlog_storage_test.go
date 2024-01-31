/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestWLogStorage_Read(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		touched := false
		events := &mockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(istructs.FirstOffset, nil))
			})
		appStructs := &mockAppStructs{}
		appStructs.On("AppDef").Return(&nilAppDef{})
		appStructs.On("Events").Return(events)
		appStructs.On("Records").Return(&nilRecords{})
		appStructs.On("ViewRecords").Return(&nilViewRecords{})
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		kb, err := s.KeyBuilder(WLog, appdef.NullQName)
		require.NoError(err)
		kb.PutInt64(Field_Offset, 1)
		kb.PutInt64(Field_Count, 1)

		require.NoError(s.Read(kb, func(key istructs.IKey, _ istructs.IStateValue) (err error) {
			touched = true
			require.Equal(int64(1), key.AsInt64(Field_Offset))
			return err
		}))

		require.True(touched)
	})
	t.Run("Should return error on read wlog", func(t *testing.T) {
		require := require.New(t)
		events := &mockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).Return(errTest)
		appStructs := &mockAppStructs{}
		appStructs.On("AppDef").Return(&nilAppDef{})
		appStructs.On("Events").Return(events)
		appStructs.On("Records").Return(&nilRecords{})
		appStructs.On("ViewRecords").Return(&nilViewRecords{})
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, err := s.KeyBuilder(WLog, appdef.NullQName)
		require.NoError(err)
		k.PutInt64(Field_Offset, 1)
		k.PutInt64(Field_Count, 1)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })

		require.ErrorIs(err, errTest)
	})
}
func TestWLogStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		event := new(mockWLogEvent)
		event.On("CUDs", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(rec istructs.ICUDRow))
			cb(new(mockCUDRow))
		})
		events := &mockEvents{}
		events.On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				cb := args.Get(4).(istructs.WLogEventsReaderCallback)
				require.NoError(cb(istructs.FirstOffset, event))
			})
		appStructs := &mockAppStructs{}
		appStructs.On("AppDef").Return(&nilAppDef{})
		appStructs.On("Events").Return(events)
		appStructs.On("Records").Return(&nilRecords{})
		appStructs.On("ViewRecords").Return(&nilViewRecords{})
		s := ProvideCommandProcessorStateFactory()(context.Background(), func() istructs.IAppStructs { return appStructs }, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, 0, nil)
		kb, err := s.KeyBuilder(WLog, appdef.NullQName)
		require.NoError(err)
		kb.PutInt64(Field_Offset, 1)
		kb.PutInt64(Field_Count, 1)

		sv, ok, err := s.CanExist(kb)
		require.NoError(err)

		require.True(ok)
		require.Equal(int64(1), sv.AsInt64(Field_Offset))

		cuds := sv.AsValue(Field_CUDs)
		cud := cuds.GetAsValue(0)

		require.Equal(1, cuds.Length())
		require.NotNil(cud)
		require.PanicsWithValue(errFieldByIndexIsNotAnObjectOrArray, func() { sv.GetAsValue(0) })
		require.PanicsWithValue(errFieldByNameIsNotAnObjectOrArray, func() { cud.AsValue(Field_CUDs) })
		require.PanicsWithValue(errCurrentValueIsNotAnArray, func() { sv.GetAsInt64(0) })
		require.PanicsWithValue(errNotImplemented, func() { sv.AsRecordID(Field_Offset) })
	})
	t.Run("Should return error when error occurred on read wlog", func(t *testing.T) {
		require := require.New(t)
		events := &mockEvents{}
		events.
			On("ReadWLog", context.Background(), istructs.WSID(1), istructs.FirstOffset, 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NoError(args.Get(4).(istructs.WLogEventsReaderCallback)(istructs.FirstOffset, nil))
			}).
			On("ReadWLog", context.Background(), istructs.WSID(1), istructs.Offset(2), 1, mock.AnythingOfType("istructs.WLogEventsReaderCallback")).
			Return(errTest)
		appStructs := &mockAppStructs{}
		appStructs.On("AppDef").Return(&nilAppDef{})
		appStructs.On("Events").Return(events)
		appStructs.On("Records").Return(&nilRecords{})
		appStructs.On("ViewRecords").Return(&nilViewRecords{})
		s := ProvideCommandProcessorStateFactory()(context.Background(), func() istructs.IAppStructs { return appStructs }, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, 0, nil)
		kb1, err := s.KeyBuilder(WLog, appdef.NullQName)
		require.NoError(err)
		kb1.PutInt64(Field_Offset, 1)
		kb1.PutInt64(Field_Count, 1)
		kb2, err := s.KeyBuilder(WLog, appdef.NullQName)
		require.NoError(err)
		kb2.PutInt64(Field_Offset, 2)
		kb2.PutInt64(Field_Count, 1)

		err = s.CanExistAll([]istructs.IStateKeyBuilder{kb1, kb2}, nil)

		require.ErrorIs(err, errTest)
	})
}
func TestWLogStorage_ToJSON(t *testing.T) {
	s := &wLogStorage{iWorkspaceFunc: func() appdef.IWorkspace { return nil }}
	require := require.New(t)
	eventError := &mockEventError{}
	eventError.
		On("ErrStr").Return("Error string").
		On("QNameFromParams").Return(istructs.QNameForError).
		On("ValidEvent").Return(false).
		On("OriginalEventBytes").Return([]byte("Love bites"))
	event := &mockWLogEvent{}
	event.
		On("QName").Return(testEvent).
		On("ArgumentObject").Return(istructs.NewNullObject()).
		On("CUDs", mock.Anything).Return(nil).
		On("RegisteredAt").Return(istructs.UnixMilli(1662972220000)).
		On("Synced").Return(true).
		On("DeviceID").Return(istructs.ConnectedDeviceID(7)).
		On("SyncedAt").Return(istructs.UnixMilli(1662972220001)).
		On("Error").Return(eventError)

	sv := &wLogValue{
		event:      event,
		offset:     123,
		toJSONFunc: s.toJSON,
	}

	json, err := sv.ToJSON()
	require.NoError(err)

	require.JSONEq(`{
								  "ArgumentObject": {},
								  "CUDs":[],
								  "DeviceID": 7,
								  "Error": {
									"ErrStr": "Error string",
									"OriginalEventBytes": "TG92ZSBiaXRlcw==",
									"QNameFromParams": "sys.Error",
									"ValidEvent": false
								  },
								  "Offset": 123,
								  "QName": "test.event",
								  "RegisteredAt": 1662972220000,
								  "Synced": true,
								  "SyncedAt": 1662972220001
								}`, json)
}
