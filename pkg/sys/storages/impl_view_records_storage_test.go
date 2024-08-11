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
	"github.com/voedger/voedger/pkg/sys"
)

func mockedStructs2(t *testing.T, addWsDescriptor bool) (*mockAppStructs, *mockViewRecords) {
	appDef := appdef.New()

	appDef.AddPackage("test", "test.com/test")

	view := appDef.AddView(testViewRecordQName1)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	view = appDef.AddView(testViewRecordQName2)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	mockWorkspaceRecord := &mockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)
	mockedRecords := &mockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	mockedViews := &mockViewRecords{}
	mockedViews.On("KeyBuilder", testViewRecordQName1).Return(newMapKeyBuilder(sys.Storage_View, testViewRecordQName1))

	if addWsDescriptor {
		wsDesc := appDef.AddCDoc(testWSDescriptorQName)
		wsDesc.AddField(field_WSKind, appdef.DataKind_bytes, false)
	}

	ws := appDef.AddWorkspace(testWSQName)
	ws.AddType(testViewRecordQName1)
	ws.AddType(testViewRecordQName2)
	ws.SetDescriptor(testWSDescriptorQName)

	app, err := appDef.Build()
	require.NoError(t, err)

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(app).
		On("AppQName").Return(testAppQName).
		On("Records").Return(mockedRecords).
		On("Events").Return(&nilEvents{}).
		On("ViewRecords").Return(mockedViews)

	return appStructs, mockedViews
}

func mockedStructs(t *testing.T) (*mockAppStructs, *mockViewRecords) {
	return mockedStructs2(t, true)
}

func TestViewRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)

		mockedStructs, mockedViews := mockedStructs(t)
		valueOnGet := &mockValue{}
		valueOnGet.On("AsString", "vk").Return("value")
		mockedViews.
			On("Get", istructs.WSID(1), mock.Anything).Return(valueOnGet, nil)

		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, nil, nil, nil, nil)
		k, e := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(e)
		k.PutInt64("pkk", 64)
		k.PutString("cck", "ccv")

		sv, ok, err := s.CanExist(k)
		require.NoError(err)

		require.True(ok)
		require.Equal("value", sv.AsString("vk"))
	})
	t.Run("Should return error on get", func(t *testing.T) {
		require := require.New(t)
		mockedStructs, mockedViews := mockedStructs(t)
		mockedViews.
			On("Get", istructs.WSID(1), mock.Anything).Return(nil, errTest)

		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, nil, nil, nil, nil)
		k, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
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
		mockedStructs, mockedViews := mockedStructs(t)
		touched := false
		mockedViews.
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.AnythingOfType("istructs.ValuesCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.NotNil(args.Get(2))
				require.NoError(args.Get(3).(istructs.ValuesCallback)(nil, nil))
			})
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, nil, nil, nil, nil)
		k, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
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
		mockedStructs, mockedViews := mockedStructs(t)
		mockedViews.
			On("KeyBuilder", testViewRecordQName1).Return(newMapKeyBuilder(sys.Storage_View, testViewRecordQName1)).
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.Anything).
			Return(errTest)
		s := ProvideQueryProcessorStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, nil, nil, nil, nil)
		k, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		err = s.Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })
		require.ErrorIs(err, errTest)
	})
}
func TestViewRecordsStorage_ApplyBatch_should_return_error_on_put_batch(t *testing.T) {
	require := require.New(t)
	mockedStructs, mockedViews := mockedStructs(t)
	mockedViews.
		On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
		On("PutBatch", istructs.WSID(1), mock.Anything).Return(errTest)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 10, 10)
	kb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
	require.NoError(err)
	_, err = s.NewValue(kb)
	require.NoError(err)
	readyToFlush, err := s.ApplyIntents()
	require.False(readyToFlush)
	require.NoError(err)
	err = s.FlushBundles()
	require.ErrorIs(err, errTest)
}

func TestViewRecordsStorage_ApplyBatch_NullWSIDGoesLast(t *testing.T) {
	require := require.New(t)

	mockedStructs, mockedViews := mockedStructs(t)

	appliedWSIDs := make([]istructs.WSID, 0)
	mockedViews.
		On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
		On("PutBatch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		appliedWSIDs = append(appliedWSIDs, args[0].(istructs.WSID))
	}).
		Return(nil)

	putViewRec := func(s IBundledHostState) {
		kb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		_, err = s.NewValue(kb)
		require.NoError(err)
	}

	putOffset := func(s IBundledHostState) {
		kb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		kb.PutInt64(sys.Storage_View_Field_WSID, int64(istructs.NullWSID))
		require.NoError(err)
		_, err = s.NewValue(kb)
		require.NoError(err)
	}

	applyAndFlush := func(s IBundledHostState) {
		readyToFlush, err := s.ApplyIntents()
		require.False(readyToFlush)
		require.NoError(err)
		err = s.FlushBundles()
		require.NoError(err)
	}

	s := ProvideAsyncActualizerStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 10, 10)
	putViewRec(s)
	putViewRec(s)
	putOffset(s)
	applyAndFlush(s)
	require.Len(appliedWSIDs, 2)
	require.Equal(istructs.NullWSID, appliedWSIDs[1])

	appliedWSIDs = appliedWSIDs[:0]
	putOffset(s)
	putViewRec(s)
	putViewRec(s)
	applyAndFlush(s)
	require.Len(appliedWSIDs, 2)
	require.Equal(istructs.NullWSID, appliedWSIDs[1])
}

func TestViewRecordsStorage_ValidateInWorkspaces(t *testing.T) {
	require := require.New(t)

	mockedStructs, mockedViews := mockedStructs(t)
	mockedViews.
		On("KeyBuilder", mock.Anything).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", mock.Anything).Return(&nilValueBuilder{}).
		On("Get", istructs.WSID(1), mock.Anything).Return(&nilValue{}, nil).
		On("PutBatch", mock.Anything, mock.Anything).Return(nil)
	s := ProvideAsyncActualizerStateFactory()(context.Background(), appStructsFunc(mockedStructs), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 10, 10)

	wrongQName := appdef.NewQName("test", "viewRecordX")
	wrongKb, err := s.KeyBuilder(sys.Storage_View, wrongQName)
	require.NoError(err)
	expectedError := typeIsNotDefinedInWorkspaceWithDescriptor(wrongQName, testWSDescriptorQName)

	t.Run("App() should return value", func(t *testing.T) {
		app := s.App()
		require.NotNil(app)
		require.Equal(testAppQName, app)
	})

	t.Run("State should supports istructs.IPkgNameResolver interface", func(t *testing.T) {
		const (
			pkgName = "test"
			pkgPath = "test.com/test"
		)
		require.Equal(pkgName, s.PackageLocalName(pkgPath))
		require.Equal(pkgPath, s.PackageFullPath(pkgName))
	})

	t.Run("NewValue should validate for unavailable views", func(t *testing.T) {
		value, err := s.NewValue(wrongKb)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("UpdateValue should validate for unavailable workspaces", func(t *testing.T) {
		value, err := s.UpdateValue(wrongKb, nil)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("CanExist should validate for unavailable views", func(t *testing.T) {
		value, ok, err := s.CanExist(wrongKb)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
		require.False(ok)
	})

	t.Run("MustExist should validate for unavailable views", func(t *testing.T) {
		value, err := s.MustExist(wrongKb)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("MustNotExist should validate for unavailable views", func(t *testing.T) {
		err := s.MustNotExist(wrongKb)
		require.EqualError(err, expectedError.Error())
	})

	t.Run("CanExistAll should validate for unavailable views", func(t *testing.T) {
		correctKb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		err = s.CanExistAll([]istructs.IStateKeyBuilder{wrongKb, correctKb}, nil)
		require.EqualError(err, expectedError.Error())
	})

	t.Run("MustExistAll should validate for unavailable views", func(t *testing.T) {
		correctKb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		err = s.MustExistAll([]istructs.IStateKeyBuilder{wrongKb, correctKb}, nil)
		require.EqualError(err, expectedError.Error())
	})

	t.Run("MustNotExistAll should validate for unavailable views", func(t *testing.T) {
		correctKb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		err = s.MustNotExistAll([]istructs.IStateKeyBuilder{wrongKb, correctKb})
		require.EqualError(err, expectedError.Error())
	})

}
