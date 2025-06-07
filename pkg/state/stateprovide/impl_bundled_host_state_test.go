/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func TestBundledHostState_BasicUsage(t *testing.T) {
	require := require.New(t)
	factory := ProvideAsyncActualizerStateFactory()
	n10nFn := func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {}

	// Create instance of async actualizer state
	aaState := factory(context.Background(), mockedAppStructs, nil, state.SimpleWSIDFunc(istructs.WSID(1)), n10nFn, nil, nil, nil, nil, 2, 1)

	// Declare simple extension
	extension := func(s istructs.IState, intents istructs.IIntents) {
		//Create key
		kb, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		kb.PutInt64("pkFld", int64(64))

		// Create new value
		eb, err := intents.NewValue(kb)
		require.NoError(err)
		eb.PutInt64("vFld", 10)
		eb.PutInt64(state.ColOffset, 45)
	}

	// Run extension
	extension(aaState, aaState)

	// Apply intents
	readyToFlush, err := aaState.ApplyIntents()
	require.NoError(err)
	require.True(readyToFlush)

	_ = aaState.FlushBundles()
}

func mockedAppStructs() istructs.IAppStructs {
	mv := &mockValue{}
	mv.
		On("AsInt64", "vFld").Return(int64(10)).
		On("AsInt64", state.ColOffset).Return(int64(45))
	mvb1 := &mockValueBuilder{}
	mvb1.
		On("PutInt64", "vFld", int64(10)).
		On("PutInt64", state.ColOffset, int64(45)).
		On("Build").Return(mv)
	mvb2 := &mockValueBuilder{}
	mvb2.
		On("PutInt64", "vFld", int64(10)).Once().
		On("PutInt64", state.ColOffset, int64(45)).Once().
		On("PutInt64", "vFld", int64(17)).Once().
		On("PutInt64", state.ColOffset, int64(46)).Once()
	mkb := &mockKeyBuilder{}
	mkb.
		On("PutInt64", "pkFld", int64(64))

	viewRecords := &mockViewRecords{}
	viewRecords.
		On("KeyBuilder", testViewRecordQName1).Return(mkb).
		On("NewValueBuilder", testViewRecordQName1).Return(mvb1).Once().
		On("NewValueBuilder", testViewRecordQName1).Return(mvb2).Once().
		On("PutBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewKV")).Return(nil)

	mockWorkspaceRecord := &mockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)
	mockedRecords := &mockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	adb := builder.New()
	wsb := adb.AddWorkspace(testWSQName)
	wsDesc := wsb.AddCDoc(testWSDescriptorQName)
	wsDesc.AddField(authnz.Field_WSKind, appdef.DataKind_bytes, false)
	wsb.SetDescriptor(testWSDescriptorQName)

	view := wsb.AddView(testViewRecordQName1)
	view.Key().PartKey().AddField("pkFld", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("ccFld", appdef.DataKind_string)
	view.Value().
		AddField("vFld", appdef.DataKind_int64, true).
		AddField(state.ColOffset, appdef.DataKind_int64, true)

	app := adb.MustBuild()

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(app).
		On("AppQName").Return(testAppQName).
		On("ViewRecords").Return(viewRecords).
		On("Events").Return(&nilEvents{}).
		On("Records").Return(mockedRecords)
	return appStructs
}

func TestAsyncActualizerState_CanExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = &mockStateValue{}
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_, ok, err := s.CanExist(kb)
		require.NoError(err)

		require.True(ok)
	})
	t.Run("Should return error when error occurred on get batch", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_, _, err = s.CanExist(kb)

		require.ErrorIs(err, errTest)
	})
}
func TestAsyncActualizerState_CanExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		times := 0
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(nil)
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		kb2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_ = s.CanExistAll([]istructs.IStateKeyBuilder{kb1, kb2}, func(istructs.IKeyBuilder, istructs.IStateValue, bool) error {
			times++
			return nil
		})

		require.Equal(2, times)
	})
	t.Run("Should return error when error occurred on can exist", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		kb2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.CanExistAll([]istructs.IStateKeyBuilder{kb1, kb2}, nil)

		require.ErrorIs(err, errTest)
	})
}
func TestAsyncActualizerState_MustExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = &mockStateValue{}
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_, err = s.MustExist(kb)

		require.NoError(err)
	})
	t.Run("Should return error when entity not exists", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = nil
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_, err = s.MustExist(kb)

		require.ErrorIs(err, ErrNotExists)
	})
	t.Run("Should return error when error occurred on can exist", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		_, err = s.MustExist(kb)

		require.ErrorIs(err, errTest)
	})
}
func TestAsyncActualizerState_MustExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = &mockStateValue{}
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		k2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		kk := make([]istructs.IKeyBuilder, 0, 2)

		_ = s.MustExistAll([]istructs.IStateKeyBuilder{k1, k2}, func(key istructs.IKeyBuilder, value istructs.IStateValue, ok bool) (err error) {
			kk = append(kk, key)
			require.True(ok)
			return
		})

		require.Equal(k1, kk[0])
		require.Equal(k1, kk[1])
	})
	t.Run("Should return error when entity not exists", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = nil
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		k2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustExistAll([]istructs.IStateKeyBuilder{k1, k2}, nil)

		require.ErrorIs(err, ErrNotExists)
	})
}
func TestAsyncActualizerState_MustNotExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = nil
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustNotExist(k)

		require.NoError(err)
	})
	t.Run("Should return error when entity exists", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = &mockStateValue{}
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustNotExist(k)

		require.ErrorIs(err, ErrExists)
	})
	t.Run("Should return error when error occurred on must exist", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		kb, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustNotExist(kb)

		require.ErrorIs(err, errTest)
	})
}
func TestAsyncActualizerState_MustNotExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = nil
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		k2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustNotExistAll([]istructs.IStateKeyBuilder{k1, k2})

		require.NoError(err)
	})
	t.Run("Should return error when entity exists", func(t *testing.T) {
		require := require.New(t)
		stateStorage := &mockStorage{}
		stateStorage.
			On("NewKeyBuilder", testViewRecordQName1, nil).Return(newMapKeyBuilder(testStorage, testViewRecordQName1)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]state.GetBatchItem)[0].Value = &mockStateValue{}
			})
		s := asyncActualizerStateWithTestStateStorage(stateStorage)
		k1, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)
		k2, err := s.KeyBuilder(testStorage, testViewRecordQName1)
		require.NoError(err)

		err = s.MustNotExistAll([]istructs.IStateKeyBuilder{k1, k2})

		require.ErrorIs(err, ErrExists)
	})
}
func TestAsyncActualizerState_Read(t *testing.T) {
	t.Run("Should flush bundle before read", func(t *testing.T) {
		require := require.New(t)
		touched := false
		mockedStructs, mockedViews := mockedStructs(t)
		mockedViews.
			On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
			On("KeyBuilder", testViewRecordQName2).Return(&nilKeyBuilder{}).
			On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
			On("NewValueBuilder", testViewRecordQName2).Return(&nilValueBuilder{}).
			On("PutBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewKV")).
			Return(nil).
			Run(func(args mock.Arguments) {
				require.Len(args.Get(1).([]istructs.ViewKV), 2)
			}).
			On("Read", context.Background(), istructs.WSID(1), mock.Anything, mock.AnythingOfType("istructs.ValuesCallback")).
			Return(nil).
			Run(func(args mock.Arguments) {
				_ = args.Get(3).(istructs.ValuesCallback)(&nilKey{}, &nilValue{})
			})

		s := ProvideAsyncActualizerStateFactory()(context.Background(), func() istructs.IAppStructs { return mockedStructs }, nil, state.SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 10, 10)
		kb1, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		kb2, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName2)
		require.NoError(err)

		_, _ = s.NewValue(kb1)
		_, _ = s.NewValue(kb2)

		readyToFlush, err := s.ApplyIntents()
		require.False(readyToFlush)
		require.NoError(err)

		_ = s.Read(kb1, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			touched = true
			return
		})

		require.True(touched)
	})
	t.Run("Should return error when error occurred on apply batch", func(t *testing.T) {
		require := require.New(t)
		touched := false
		mockedStructs, mockedViews := mockedStructs(t)
		mockedViews.
			On("KeyBuilder", testViewRecordQName1).Return(&nilKeyBuilder{}).
			On("KeyBuilder", testViewRecordQName2).Return(&nilKeyBuilder{}).
			On("NewValueBuilder", testViewRecordQName1).Return(&nilValueBuilder{}).
			On("NewValueBuilder", testViewRecordQName2).Return(&nilValueBuilder{}).
			On("PutBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewKV")).Return(errTest)
		s := ProvideAsyncActualizerStateFactory()(context.Background(), func() istructs.IAppStructs { return mockedStructs }, nil, state.SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil, nil, nil, 10, 10)
		kb1, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName1)
		require.NoError(err)
		kb2, err := s.KeyBuilder(sys.Storage_View, testViewRecordQName2)
		require.NoError(err)

		_, _ = s.NewValue(kb1)
		_, _ = s.NewValue(kb2)

		readyToFlush, err := s.ApplyIntents()
		require.False(readyToFlush)
		require.NoError(err)

		err = s.Read(kb1, func(key istructs.IKey, value istructs.IStateValue) (err error) {
			touched = true
			return err
		})

		require.ErrorIs(err, errTest)
		require.False(touched)
	})
}
func asyncActualizerStateWithTestStateStorage(s *mockStorage) istructs.IState {
	as := ProvideAsyncActualizerStateFactory()(context.Background(), nilAppStructsFunc, nil, nil, nil, nil, nil, nil, nil, 10, 10)
	as.(*asyncActualizerState).addStorage(testStorage, s, S_GET_BATCH|S_READ)
	return as
}
