/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package storages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

func TestViewRecordsStorage_GetBatch(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)

		mockedStructs, mockedViews := mockedStructs(t)
		valueOnGet := &mockValue{}
		valueOnGet.On("AsString", "vk").Return("value")
		mockedViews.
			On("Get", istructs.WSID(1), mock.Anything).Return(valueOnGet, nil)

		appStructsFunc := func() istructs.IAppStructs {
			return mockedStructs
		}
		storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(testViewRecordQName1, nil)
		k.PutInt64("pkk", 64)
		k.PutString("cck", "ccv")

		sv, err := storage.(state.IWithGet).Get(k)
		require.NoError(err)
		require.NotNil(sv)
		require.Equal("value", sv.AsString("vk"))
	})
	t.Run("Should return error on get", func(t *testing.T) {
		require := require.New(t)
		mockedStructs, mockedViews := mockedStructs(t)
		mockedViews.
			On("Get", istructs.WSID(1), mock.Anything).Return(nil, errTest)

		appStructsFunc := func() istructs.IAppStructs {
			return mockedStructs
		}
		storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(testViewRecordQName1, nil)
		k.PutInt64("pkk", 64)
		sv, err := storage.(state.IWithGet).Get(k)
		require.Nil(sv)
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
		appStructsFunc := func() istructs.IAppStructs {
			return mockedStructs
		}
		storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(testViewRecordQName1, nil)

		err := storage.(state.IWithRead).Read(k, func(istructs.IKey, istructs.IStateValue) error {
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
			On("KeyBuilder", testViewRecordQName1).Return(newUniqKeyBuilder(sys.Storage_View, testViewRecordQName1)).
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.Anything).
			Return(errTest)

		appStructsFunc := func() istructs.IAppStructs {
			return mockedStructs
		}
		storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
		k := storage.NewKeyBuilder(testViewRecordQName1, nil)
		err := storage.(state.IWithRead).Read(k, func(istructs.IKey, istructs.IStateValue) error { return nil })
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
	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)
	kb := storage.NewKeyBuilder(testViewRecordQName1, nil)
	vb, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
	require.NoError(err)
	err = storage.(state.IWithApplyBatch).ApplyBatch([]state.ApplyBatchItem{{Key: kb, Value: vb}})
	require.ErrorIs(err, errTest)
}

func TestViewRecordsStorage_ApplyBatch_NullWSIDGoesLast(t *testing.T) {
	require := require.New(t)

	mockedStructs, mockedViews := mockedStructs(t)

	appliedWSIDs := make([]istructs.WSID, 0)
	batch := make([]state.ApplyBatchItem, 0)

	mockedViews.
		On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
		On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
		On("PutBatch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		appliedWSIDs = append(appliedWSIDs, args[0].(istructs.WSID))
	}).
		Return(nil)

	putViewRec := func(s state.IStateStorage) {
		kb := s.NewKeyBuilder(testViewRecordQName1, nil)
		vb, err := s.(state.IWithInsert).ProvideValueBuilder(kb, nil)
		require.NoError(err)
		batch = append(batch, state.ApplyBatchItem{Key: kb, Value: vb})
	}

	putOffset := func(s state.IStateStorage) {
		kb := s.NewKeyBuilder(testViewRecordQName1, nil)
		kb.PutInt64(sys.Storage_View_Field_WSID, int64(istructs.NullWSID))
		vb, err := s.(state.IWithInsert).ProvideValueBuilder(kb, nil)
		require.NoError(err)
		batch = append(batch, state.ApplyBatchItem{Key: kb, Value: vb})
	}

	applyAndFlush := func(s state.IStateStorage) {
		err := s.(state.IWithApplyBatch).ApplyBatch(batch)
		require.NoError(err)
		batch = batch[:0]
	}

	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	s := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)

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

	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)

	wrongQName := appdef.NewQName("test", "viewRecordX")
	wrongKb := storage.NewKeyBuilder(wrongQName, nil)
	expectedError := typeIsNotDefinedInWorkspaceWithDescriptor(wrongQName, testWSDescriptorQName)

	t.Run("NewValue should validate for unavailable views", func(t *testing.T) {
		value, err := storage.(state.IWithInsert).ProvideValueBuilder(wrongKb, nil)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("UpdateValue should validate for unavailable workspaces", func(t *testing.T) {
		value, err := storage.(state.IWithUpdate).ProvideValueBuilderForUpdate(wrongKb, nil, nil)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("Get should validate for unavailable views", func(t *testing.T) {
		value, err := storage.(state.IWithGet).Get(wrongKb)
		require.EqualError(err, expectedError.Error())
		require.Nil(value)
	})

	t.Run("GetBatch should validate for unavailable views", func(t *testing.T) {
		correctKb := storage.NewKeyBuilder(testViewRecordQName1, nil)
		err := storage.(state.IWithGetBatch).GetBatch([]state.GetBatchItem{{Key: wrongKb}, {Key: correctKb}})
		require.EqualError(err, expectedError.Error())
	})

}

func TestViewRecordsStorage_PutInt64ForRecordIDFields(t *testing.T) {
	require := require.New(t)
	fieldName1 := "i64"
	fieldName2 := "recID"
	value1 := int64(1)
	value2 := int64(2)

	mockedValueBuilder := &mockValueBuilder{}
	mockedValueBuilder.On("PutInt64", fieldName1, value1)
	mockedValueBuilder.On("PutRecordID", fieldName2, istructs.RecordID(value2))

	mockedStructs, mockedViews := mockedStructs(t)
	mockedViews.
		On("NewValueBuilder", mock.Anything).Return(mockedValueBuilder)

	appStructsFunc := func() istructs.IAppStructs {
		return mockedStructs
	}
	storage := NewViewRecordsStorage(context.Background(), appStructsFunc, state.SimpleWSIDFunc(istructs.WSID(1)), nil)

	kb := storage.NewKeyBuilder(testViewRecordQName1, nil)
	value, err := storage.(state.IWithInsert).ProvideValueBuilder(kb, nil)
	require.NoError(err)
	value.PutInt64(fieldName1, value1)
	value.PutInt64(fieldName2, value2)
	mockedValueBuilder.AssertExpectations(t)
}
