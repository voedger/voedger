/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func createAppDef() appdef.IAppDef {
	adb := builder.New()

	wsb := adb.AddWorkspace(testWSQName)
	wsDesc := wsb.AddCDoc(testWSDescriptorQName)
	wsDesc.AddField(field_WSKind, appdef.DataKind_bytes, false)
	wsDesc.SetSingleton()
	wsb.SetDescriptor(testWSDescriptorQName)

	wsb.AddObject(testRecordQName1).
		AddField("number", appdef.DataKind_int64, false)
	wsb.AddObject(testRecordQName2).
		AddField("age", appdef.DataKind_int64, false).
		AddField("ref", appdef.DataKind_RecordID, false)

	return adb.MustBuild()
}

func TestRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should handle general records", func(t *testing.T) {
		require := require.New(t)
		records := &mockRecords{}
		records.
			On("GetBatch", istructs.WSID(1), true, mock.AnythingOfType("[]istructs.RecordGetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				items := args.Get(2).([]istructs.RecordGetBatchItem)
				record := &mockRecord{}
				record.On("QName").Return(testRecordQName1)
				record.On("AsInt64", "number").Return(int64(10))
				items[0].Record = record
			}).
			On("GetBatch", istructs.WSID(2), true, mock.AnythingOfType("[]istructs.RecordGetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				items := args.Get(2).([]istructs.RecordGetBatchItem)
				record1 := &mockRecord{}
				record1.
					On("QName").Return(testRecordQName2).
					On("AsInt64", "age").Return(int64(20))
				items[0].Record = record1
				record2 := &mockRecord{}
				record2.
					On("QName").Return(testRecordQName2).
					On("AsInt64", "age").Return(int64(21))
				items[1].Record = record2
			})

		adb := builder.New()
		wsb := adb.AddWorkspace(testWSQName)
		wsb.AddObject(testRecordQName1).
			AddField("number", appdef.DataKind_int64, false)
		wsb.AddObject(testRecordQName2).
			AddField("age", appdef.DataKind_int64, false)

		app, err := adb.Build()
		require.NoError(err)

		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(app).
			On("AppQName").Return(testAppQName).
			On("Records").Return(records).
			On("ViewRecords").Return(&nilViewRecords{}).
			On("Events").Return(&nilEvents{})

		appStructsFunc := func() istructs.IAppStructs {
			return appStructs
		}
		storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k1 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k1.PutRecordID(sys.Storage_Record_Field_ID, 1)
		k2 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k2.PutRecordID(sys.Storage_Record_Field_ID, 2)
		k2.PutInt64(sys.Storage_Record_Field_WSID, 2)
		k3 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k3.PutRecordID(sys.Storage_Record_Field_ID, 3)
		k3.PutInt64(sys.Storage_Record_Field_WSID, 2)

		batchItems := []state.GetBatchItem{
			{Key: k1},
			{Key: k2},
			{Key: k3},
		}
		err = storage.(state.IWithGetBatch).GetBatch(batchItems)
		require.NoError(err)
		require.Equal(int64(10), batchItems[0].Value.AsInt64("number"))
		require.Equal(int64(20), batchItems[1].Value.AsInt64("age"))
		require.Equal(int64(21), batchItems[2].Value.AsInt64("age"))
	})
	t.Run("Should handle singleton records", func(t *testing.T) {
		require := require.New(t)
		singleton1 := &mockRecord{}
		singleton1.
			On("QName").Return(testRecordQName1).
			On("AsQName", mock.Anything).Return(testRecordQName1).
			On("AsInt64", "number").Return(int64(10)).
			On("FieldNames", mock.Anything).Run(func(a mock.Arguments) {
			x := a.Get(0).(func(name string))
			x("number")
		})
		singleton2 := &mockRecord{}
		singleton2.
			On("QName").Return(testRecordQName2).
			On("AsQName", mock.Anything).Return(testRecordQName2).
			On("AsInt64", "age").Return(int64(18)).
			On("FieldNames", mock.Anything).Run(func(a mock.Arguments) {
			x := a.Get(0).(func(name string))
			x("age")
		})
		nullRecord := &mockRecord{}
		nullRecord.On("QName").Return(appdef.NullQName)

		mockWorkspaceRecord := &mockRecord{}
		mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
		mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)

		records := &mockRecords{}
		records.
			On("GetSingleton", istructs.WSID(1), testRecordQName1).Return(singleton1, nil).
			On("GetSingleton", istructs.WSID(2), testRecordQName2).Return(nullRecord, nil).
			On("GetSingleton", istructs.WSID(3), testRecordQName2).Return(singleton2, nil).
			On("GetSingleton", mock.Anything, qNameCDocWorkspaceDescriptor).Return(mockWorkspaceRecord, nil)

		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(createAppDef()).
			On("AppQName").Return(testAppQName).
			On("Records").Return(records).
			On("ViewRecords").Return(&nilViewRecords{}).
			On("Events").Return(&nilEvents{})

		appStructsFunc := func() istructs.IAppStructs {
			return appStructs
		}
		storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k1 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k1.PutQName(sys.Storage_Record_Field_Singleton, testRecordQName1)
		k2 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k2.PutQName(sys.Storage_Record_Field_Singleton, testRecordQName2)
		k2.PutInt64(sys.Storage_Record_Field_WSID, 2)
		k3 := storage.NewKeyBuilder(appdef.NullQName, nil)
		k3.PutQName(sys.Storage_Record_Field_Singleton, testRecordQName2)
		k3.PutInt64(sys.Storage_Record_Field_WSID, 3)
		k4 := storage.NewKeyBuilder(qNameCDocWorkspaceDescriptor, nil)
		k4.PutBool(sys.Storage_Record_Field_IsSingleton, true)
		batchItems := []state.GetBatchItem{
			{Key: k1},
			{Key: k2},
			{Key: k3},
			{Key: k4},
		}
		err := storage.(state.IWithGetBatch).GetBatch(batchItems)
		require.NoError(err)
		require.Equal(int64(10), batchItems[0].Value.AsInt64("number"))
		require.Equal(int64(18), batchItems[2].Value.AsInt64("age"))
	})
	t.Run("Should return error when 'id' not found", func(t *testing.T) {
		require := require.New(t)
		appStructs := &mockAppStructs{}
		appStructsFunc := func() istructs.IAppStructs {
			return appStructs
		}
		storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(appdef.NullQName, nil)
		_, err := storage.(state.IWithGet).Get(k)
		require.ErrorIs(err, ErrNotFound)
	})
	t.Run("Should return error on get", func(t *testing.T) {
		require := require.New(t)
		records := &mockRecords{}
		records.On("Get", istructs.WSID(1), true, mock.Anything).Return(nil, errTest)
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(createAppDef()).
			On("AppQName").Return(testAppQName).
			On("Records").Return(records).
			On("ViewRecords").Return(&nilViewRecords{}).
			On("Events").Return(&nilEvents{})
		appStructsFunc := func() istructs.IAppStructs {
			return appStructs
		}
		storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(appdef.NullQName, nil)
		k.PutRecordID(sys.Storage_Record_Field_ID, istructs.RecordID(1))
		_, err := storage.(state.IWithGet).Get(k)
		require.ErrorIs(err, errTest)
	})
	t.Run("Should return error on get singleton", func(t *testing.T) {
		require := require.New(t)

		mockWorkspaceRecord := &mockRecord{}
		mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
		mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)

		records := &mockRecords{}
		records.On("GetSingleton", istructs.WSID(1), testRecordQName1).Return(&mockRecord{}, errTest)
		records.On("GetSingleton", mock.Anything, qNameCDocWorkspaceDescriptor).Return(mockWorkspaceRecord, nil)
		appStructs := &mockAppStructs{}
		appStructs.
			On("AppDef").Return(createAppDef()).
			On("AppQName").Return(testAppQName).
			On("Records").Return(records).
			On("ViewRecords").Return(&nilViewRecords{}).
			On("Events").Return(&nilEvents{})
		appStructsFunc := func() istructs.IAppStructs {
			return appStructs
		}
		storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(appdef.NullQName, nil)
		k.PutQName(sys.Storage_Record_Field_Singleton, testRecordQName1)
		_, err := storage.(state.IWithGet).Get(k)
		require.ErrorIs(err, errTest)
	})
}
func TestRecordsStorage_Insert(t *testing.T) {
	require := require.New(t)
	fieldName := "name"
	value := "Voedger"
	rw := &mockRowWriter{}
	rw.
		On("PutString", fieldName, value)
	cud := &mockCUD{}
	cud.On("Create").Return(rw)

	records := &mockRecords{}

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(createAppDef()).
		On("AppQName").Return(testAppQName).
		On("Records").Return(records).
		On("ViewRecords").Return(&nilViewRecords{}).
		On("Events").Return(&nilEvents{})
	appStructsFunc := func() istructs.IAppStructs {
		return appStructs
	}
	cudFunc := func() istructs.ICUD {
		return cud
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.NullWSID), cudFunc)
	kb := storage.NewKeyBuilder(testRecordQName1, nil)
	vb, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
	require.NoError(err)
	vb.PutString(fieldName, value)
	rw.AssertExpectations(t)
}
func TestRecordsStorage_InsertRecordIDField(t *testing.T) {
	require := require.New(t)
	fieldName1 := "ref"
	fieldName2 := "age"
	var value1 int64 = 1234
	var value2 int64 = 18

	rw := &mockRowWriter{}
	rw.
		On("PutRecordID", fieldName1, istructs.RecordID(value1))
	rw.
		On("PutInt64", fieldName2, value2)
	cud := &mockCUD{}
	cud.On("Create").Return(rw)

	records := &mockRecords{}

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(createAppDef()).
		On("AppQName").Return(testAppQName).
		On("Records").Return(records).
		On("ViewRecords").Return(&nilViewRecords{}).
		On("Events").Return(&nilEvents{})
	appStructsFunc := func() istructs.IAppStructs {
		return appStructs
	}
	cudFunc := func() istructs.ICUD {
		return cud
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.NullWSID), cudFunc)
	kb := storage.NewKeyBuilder(testRecordQName2, nil)
	vb, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
	require.NoError(err)
	vb.PutInt64(fieldName1, value1)
	vb.PutInt64(fieldName2, value2)
	rw.AssertExpectations(t)
}
func TestRecordsStorage_Update(t *testing.T) {
	require := require.New(t)
	fieldName := "name"
	value := "Voedger"
	rw := &mockRowWriter{}
	rw.On("PutString", fieldName, value)
	r := &mockRecord{}
	r.On("QName").Return(appdef.NewQName("test", "Record1"))
	sv := &recordsValue{record: r}
	cud := &mockCUD{}
	cud.On("Update", mock.Anything).Return(rw)
	cudFunc := func() istructs.ICUD {
		return cud
	}
	appStructs := &mockAppStructs{}
	appStructsFunc := func() istructs.IAppStructs {
		return appStructs
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.NullWSID), cudFunc)
	kb := storage.NewKeyBuilder(testRecordQName1, nil)
	vb, err := storage.(state.IWithUpdate).ProvideValueBuilderForUpdate(kb, sv, nil)
	require.NoError(err)
	vb.PutString(fieldName, value)
	rw.AssertExpectations(t)
}

func TestRecordsStorage_ValidateInWorkspaces_Reads(t *testing.T) {
	require := require.New(t)

	mockedStructs, mockedViews := mockedStructs(t)
	mockedViews.
		On("KeyBuilder", mock.Anything).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", mock.Anything).Return(&nilValueBuilder{}).
		On("Get", istructs.WSID(1), mock.Anything).Return(&nilValue{}, nil).
		On("PutBatch", mock.Anything, mock.Anything).Return(nil)

	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)

	wrongSingleton := appdef.NewQName("test", "RecordX")
	wrongKb := storage.NewKeyBuilder(appdef.NullQName, nil)
	wrongKb.PutQName(sys.Storage_Record_Field_Singleton, wrongSingleton)
	expectedError := typeIsNotDefinedInWorkspaceWithDescriptor(wrongSingleton, testWSDescriptorQName)
	var err error

	t.Run("Get should validate for unavailable records", func(t *testing.T) {
		value, err := storage.(state.IWithGet).Get(wrongKb)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("CanExistAll should validate for unavailable records", func(t *testing.T) {
		err = storage.(state.IWithGetBatch).GetBatch([]state.GetBatchItem{{Key: wrongKb}})
		require.EqualError(err, expectedError.Error())
	})
}

func TestRecordsStorage_ValidateInWorkspaces_Writes(t *testing.T) {
	require := require.New(t)

	mockedStructs, mockedViews := mockedStructs(t)
	mockedViews.
		On("KeyBuilder", mock.Anything).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", mock.Anything).Return(&nilValueBuilder{}).
		On("Get", istructs.WSID(1), mock.Anything).Return(&nilValue{}, nil).
		On("PutBatch", mock.Anything, mock.Anything).Return(nil)

	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewRecordsStorage(appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)

	wrongSingleton := appdef.NewQName("test", "RecordX")
	wrongKb := storage.NewKeyBuilder(wrongSingleton, nil)
	expectedError := typeIsNotDefinedInWorkspaceWithDescriptor(wrongSingleton, testWSDescriptorQName)

	t.Run("NewValue should validate for unavailable records", func(t *testing.T) {
		builder, err := storage.(state.IWithInsert).ProvideValueBuilder(wrongKb, nil)
		require.EqualError(err, expectedError.Error())
		require.Nil(builder)
	})

}
