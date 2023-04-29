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
	amock "github.com/voedger/voedger/pkg/appdef/mock"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestViewRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)

		view := amock.NewView(testViewRecordQName1)
		view.
			AddPartField("pkk", appdef.DataKind_string). // ??? variable len PK !!!
			AddClustColumn("cck", appdef.DataKind_string).
			AddValueField("vk", appdef.DataKind_string, false)

		appDef := amock.NewAppDef()
		appDef.AddView(view)

		value := &mockValue{}
		value.On("AsString", "vk").Return("value")
		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("GetBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewRecordGetBatchItem")).Return(nil).
			Run(func(args mock.Arguments) {
				items := args.Get(1).([]istructs.ViewRecordGetBatchItem)
				items[0].Ok = true
				items[0].Value = value
			})
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(appDef).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, e := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.Nil(e)
		k.PutString("pkk", "pkv")
		k.PutString("cck", "ccv")

		sv, ok, err := s.CanExist(k)
		require.NoError(err)

		require.True(ok)
		require.Equal("value", sv.AsString("vk"))
	})
	t.Run("Should return error on get batch", func(t *testing.T) {
		require := require.New(t)

		appDef := amock.NewAppDef()
		appDef.AddView(amock.NewView(testViewRecordQName1))

		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("GetBatch", istructs.WSID(1), mock.Anything).Return(errTest)
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(appDef).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, err := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.NoError(err)
		k.PutString("pkk", "pkv")

		_, ok, err := s.CanExist(k)

		require.False(ok)
		require.ErrorIs(err, errTest)
	})
}
func TestViewRecordsStorage_Read(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		touched := false
		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.AnythingOfType("istructs.ValuesCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NotNil(args.Get(2))
				require.NoError(args.Get(3).(istructs.ValuesCallback)(nil, nil))
			})
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(&nilAppDef{}).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, err := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error {
			touched = true
			return nil
		})
		require.NoError(err)

		require.True(touched)
	})
	t.Run("Should return error on read", func(t *testing.T) {
		require := require.New(t)
		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.Anything).
			Return(errTest)
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(&nilAppDef{}).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, err := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })

		require.ErrorIs(err, errTest)
	})
}
func TestViewRecordsStorage_ApplyBatch_should_return_error_on_put_batch(t *testing.T) {
	require := require.New(t)

	appDef := amock.NewAppDef()
	appDef.AddView(amock.NewView(testViewRecordQName1))

	viewRecords := &mockViewRecords{}
	viewRecords.
		On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
		On("PutBatch", istructs.WSID(1), mock.Anything).Return(errTest)
	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(appDef).
		On("ViewRecords").Return(viewRecords).
		On("Records").Return(&nilRecords{}).
		On("Events").Return(&nilEvents{})
	s := ProvideAsyncActualizerStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, 10, 10)
	kb, err := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
	require.NoError(err)
	_, err = s.NewValue(kb)
	require.NoError(err)
	readyToFlush, err := s.ApplyIntents()
	require.False(readyToFlush)
	require.NoError(err)

	err = s.FlushBundles()

	require.ErrorIs(err, errTest)
}

func TestViewRecordsStorage_toJSON(t *testing.T) {

	view := amock.NewView(testViewRecordQName1)
	view.
		AddPartField("pkFld", appdef.DataKind_int64).
		AddClustColumn("ccFld", appdef.DataKind_string).
		AddValueField("ID", appdef.DataKind_RecordID, false).
		AddValueField("Name", appdef.DataKind_string, false).
		AddValueField("Count", appdef.DataKind_int64, false)

	appDef := amock.NewAppDef()
	appDef.AddView(view)

	value := &mockValue{}
	value.
		On("AsRecordID", "ID").Return(istructs.RecordID(42)).
		On("AsString", "Name").Return("John").
		On("AsInt64", "Count").Return(int64(1001)).
		On("AsQName", mock.Anything).Return(appdef.ViewValueDefName(testViewRecordQName1))

	s := viewRecordsStorage{
		appDefFunc: func() appdef.IAppDef { return appDef },
	}
	t.Run("Should marshal entire element", func(t *testing.T) {
		require := require.New(t)
		sv := &viewRecordsStorageValue{
			value:      value,
			toJSONFunc: s.toJSON,
		}

		json, err := sv.ToJSON()
		require.NoError(err)

		require.JSONEq(`{
								  "Count": 1001,
								  "ID": 42,
								  "Name": "John"
								}`, json)
	})
	t.Run("Should filter fields", func(t *testing.T) {
		require := require.New(t)
		sv := &viewRecordsStorageValue{
			value:      value,
			toJSONFunc: s.toJSON,
		}

		json, err := sv.ToJSON(WithExcludeFields("ID", "Count"))
		require.NoError(err)

		require.JSONEq(`{"Name": "John"}`, json)
	})
}
