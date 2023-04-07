/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"testing"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestViewRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		pkSchema := &mockSchema{}
		pkSchema.
			On("Fields", mock.Anything).
			Run(func(args mock.Arguments) {
				cb := args.Get(0).(func(fieldName string, kind istructs.DataKindType))
				cb("pkk", istructs.DataKind_string)
			})
		ccSchema := &mockSchema{}
		ccSchema.
			On("Fields", mock.Anything).
			Run(func(args mock.Arguments) {
				cb := args.Get(0).(func(fieldName string, kind istructs.DataKindType))
				cb("cck", istructs.DataKind_string)
			})
		vSchema := &mockSchema{}
		vSchema.
			On("Fields", mock.Anything).
			Run(func(args mock.Arguments) {
				cb := args.Get(0).(func(fieldName string, kind istructs.DataKindType))
				cb("vk", istructs.DataKind_string)
			})
		schema := &mockSchema{}
		schema.
			On("Containers", mock.AnythingOfType("func(string, istructs.QName)")).
			Run(func(args mock.Arguments) {
				cb := args.Get(0).(func(string, istructs.QName))
				cb(istructs.SystemContainer_ViewPartitionKey, testViewRecordPkQName)
				cb(istructs.SystemContainer_ViewClusteringCols, testViewRecordCcQName)
				cb(istructs.SystemContainer_ViewValue, testViewRecordVQName)
			})
		schemas := &mockSchemas{}
		schemas.
			On("Schema", testViewRecordQName1).Return(schema).
			On("Schema", testViewRecordPkQName).Return(pkSchema).
			On("Schema", testViewRecordCcQName).Return(ccSchema).
			On("Schema", testViewRecordVQName).Return(vSchema)
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
			On("Schemas").Return(schemas).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil)
		k, e := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.Nil(e)
		k.PutString("pkk", "pkv")
		k.PutString("cck", "ccv")

		sv, ok, _ := s.CanExist(k)

		require.True(ok)
		require.Equal("value", sv.AsString("vk"))
	})
	t.Run("Should return error on get batch", func(t *testing.T) {
		require := require.New(t)
		schema := &mockSchema{}
		schema.On("Containers", mock.Anything)
		schemas := &mockSchemas{}
		schemas.On("Schema", testViewRecordQName1).Return(schema)
		viewRecords := &mockViewRecords{}
		viewRecords.
			On("KeyBuilder", testViewRecordQName1).Return(newKeyBuilder(ViewRecordsStorage, testViewRecordQName1)).
			On("GetBatch", istructs.WSID(1), mock.Anything).Return(errTest)
		appStructs := &mockAppStructs{}
		appStructs.
			On("Schemas").Return(schemas).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil)
		k, _ := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
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
				_ = args.Get(3).(istructs.ValuesCallback)(nil, nil)
			})
		appStructs := &mockAppStructs{}
		appStructs.
			On("Schemas").Return(&nilSchemas{}).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil)
		k, _ := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)

		_ = s.Read(k, func(istructs.IKey, istructs.IStateValue) error {
			touched = true
			return nil
		})

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
			On("Schemas").Return(&nilSchemas{}).
			On("Records").Return(&nilRecords{}).
			On("Events").Return(&nilEvents{}).
			On("ViewRecords").Return(viewRecords)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil)
		k, _ := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)

		err := s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })

		require.ErrorIs(err, errTest)
	})
}
func TestViewRecordsStorage_ApplyBatch_should_return_error_on_put_batch(t *testing.T) {
	require := require.New(t)
	schemas := &mockSchemas{}
	schemas.On("Schema", testViewRecordQName1).Return(&nilSchema{})
	viewRecords := &mockViewRecords{}
	viewRecords.
		On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
		On("PutBatch", istructs.WSID(1), mock.Anything).Return(errTest)
	appStructs := &mockAppStructs{}
	appStructs.
		On("ViewRecords").Return(viewRecords).
		On("Schemas").Return(schemas).
		On("Records").Return(&nilRecords{}).
		On("Events").Return(&nilEvents{})
	s := ProvideAsyncActualizerStateFactory()(context.Background(), appStructs, nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, 10, 10)
	kb, _ := s.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
	_, _ = s.NewValue(kb)
	readyToFlush, err := s.ApplyIntents()
	require.False(readyToFlush)
	require.Nil(err)

	err = s.FlushBundles()

	require.ErrorIs(err, errTest)
}

func TestViewRecordsStorage_toJSON(t *testing.T) {
	vSchema := &mockSchema{}
	vSchema.
		On("Fields", mock.Anything).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(fieldName string, kind istructs.DataKindType))
			cb("ID", istructs.DataKind_RecordID)
			cb("Name", istructs.DataKind_string)
			cb("Count", istructs.DataKind_int64)
		})
	schema := &mockSchema{}
	schema.
		On("Containers", mock.AnythingOfType("func(string, istructs.QName)")).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(string, istructs.QName))
			cb(istructs.SystemContainer_ViewPartitionKey, testViewRecordPkQName)
			cb(istructs.SystemContainer_ViewClusteringCols, testViewRecordCcQName)
			cb(istructs.SystemContainer_ViewValue, testViewRecordVQName)
		}).
		On("Fields", mock.Anything).
		Run(func(args mock.Arguments) {
			cb := args.Get(0).(func(fieldName string, kind istructs.DataKindType))
			cb("ID", istructs.DataKind_RecordID)
			cb("Name", istructs.DataKind_string)
			cb("Count", istructs.DataKind_int64)
		})

	schemas := &mockSchemas{}
	schemas.
		On("Schema", testViewRecordQName1).Return(schema).
		On("Schema", testViewRecordVQName).Return(vSchema)

	value := &mockValue{}
	value.
		On("AsRecordID", "ID").Return(istructs.RecordID(42)).
		On("AsString", "Name").Return("John").
		On("AsInt64", "Count").Return(int64(1001)).
		On("AsQName", mock.Anything).Return(testViewRecordQName1)

	s := viewRecordsStorage{
		schemasFunc: func() istructs.ISchemas { return schemas },
	}
	t.Run("Should marshal entire element", func(t *testing.T) {
		require := require.New(t)
		sv := &viewRecordsStorageValue{
			value:      value,
			toJSONFunc: s.toJSON,
		}

		json, _ := sv.ToJSON()

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

		json, _ := sv.ToJSON(WithExcludeFields("ID", "Count"))

		require.JSONEq(`{"Name": "John"}`, json)
	})
}
