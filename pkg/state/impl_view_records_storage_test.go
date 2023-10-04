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

func TestViewRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)

		appDef := appdef.New()
		view := appDef.AddView(testViewRecordQName1)
		view.Key().Partition().AddField("pkk", appdef.DataKind_int64)
		view.Key().ClustCols().AddStringField("cck", appdef.DefaultFieldMaxLength)
		view.Value().AddStringField("vk", false)

		value := &mockValue{}
		value.On("AsString", "vk").Return("value")
		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("Get", istructs.WSID(1), mock.Anything).Return(value, nil)

		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(appDef).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, e := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.Nil(e)
		k.PutInt64("pkk", 64)
		k.PutString("cck", "ccv")

		sv, ok, err := s.CanExist(k)
		require.NoError(err)

		require.True(ok)
		require.Equal("value", sv.AsString("vk"))
	})
	t.Run("Should return error on get", func(t *testing.T) {
		require := require.New(t)

		appDef := appdef.New()

		view := appDef.AddView(testViewRecordQName1)
		view.Key().Partition().AddField("pkk", appdef.DataKind_int64)
		view.Key().ClustCols().AddStringField("cck", appdef.DefaultFieldMaxLength)
		view.Value().AddStringField("vk", false)

		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("Get", istructs.WSID(1), mock.Anything).Return(nil, errTest)
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(appDef).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)
		k, err := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.NoError(err)
		k.PutInt64("pkk", 64)

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

	appDef := appdef.New()

	view := appDef.AddView(testViewRecordQName1)
	view.Key().Partition().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddStringField("cck", appdef.DefaultFieldMaxLength)
	view.Value().AddStringField("vk", false)

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

	appDef := appdef.New()

	view := appDef.AddView(testViewRecordQName1)
	view.Key().Partition().AddField("pkFld", appdef.DataKind_int64)
	view.Key().ClustCols().AddStringField("ccFld", appdef.DefaultFieldMaxLength)
	view.Value().
		AddRefField("ID", false).
		AddStringField("Name", false).
		AddField("Count", appdef.DataKind_int64, false)

	value := &mockValue{}
	value.
		On("AsRecordID", "ID").Return(istructs.RecordID(42)).
		On("AsString", "Name").Return("John").
		On("AsInt64", "Count").Return(int64(1001)).
		On("AsQName", mock.Anything).Return(testViewRecordQName1)

}
