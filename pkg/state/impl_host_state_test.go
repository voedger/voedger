/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
	smock "github.com/voedger/voedger/pkg/schemas/mock"
)

func TestHostState_BasicUsage(t *testing.T) {
	require := require.New(t)

	factory := ProvideQueryProcessorStateFactory()
	hostState := factory(context.Background(), mockedHostStateStructs(), nil, SimpleWSIDFunc(istructs.WSID(1)), nil, nil, nil)

	// Declare simple extension
	extension := func(state istructs.IState) {
		//Create key
		key, err := state.KeyBuilder(ViewRecordsStorage, testViewRecordQName1)
		require.NoError(err)
		key.PutString("pkFld", "pkVal")

		// Call to storage
		_ = state.MustNotExist(key)
	}

	// Run extension
	extension(hostState)

	_ = hostState.ValidateIntents()
	_ = hostState.ApplyIntents()
}

func mockedHostStateStructs() istructs.IAppStructs {
	mv := &mockValue{}
	mv.
		On("AsInt64", "vFld").Return(int64(10)).
		On("AsInt64", ColOffset).Return(int64(45))
	mvb1 := &mockValueBuilder{}
	mvb1.
		On("PutInt64", "vFld", int64(10)).
		On("PutInt64", ColOffset, int64(45)).
		On("Build").Return(mv)
	mvb2 := &mockValueBuilder{}
	mvb2.
		On("PutInt64", "vFld", int64(10)).Once().
		On("PutInt64", ColOffset, int64(45)).Once().
		On("PutInt64", "vFld", int64(17)).Once().
		On("PutInt64", ColOffset, int64(46)).Once()
	mkb := &mockKeyBuilder{}
	mkb.
		On("PutString", "pkFld", "pkVal")
	viewRecords := &mockViewRecords{}
	viewRecords.
		On("KeyBuilder", testViewRecordQName1).Return(mkb).
		On("NewValueBuilder", testViewRecordQName1).Return(mvb1).Once().
		On("NewValueBuilder", testViewRecordQName1).Return(mvb2).Once().
		On("GetBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewRecordGetBatchItem")).
		Return(nil).
		Run(func(args mock.Arguments) {
			value := &mockValue{}
			value.On("AsString", "vk").Return("value")
			args.Get(1).([]istructs.ViewRecordGetBatchItem)[0].Value = value
		}).
		On("PutBatch", istructs.WSID(1), mock.AnythingOfType("[]istructs.ViewKV")).Return(nil)

	pkSchema := smock.MockedSchema(testViewRecordPkQName, schemas.SchemaKind_ViewRecord_PartitionKey,
		smock.MockedField("pkFld", schemas.DataKind_string, true),
	)

	valSchema := smock.MockedSchema(testViewRecordVQName, schemas.SchemaKind_ViewRecord_Value,
		smock.MockedField("vFld", schemas.DataKind_int64, false),
		smock.MockedField(ColOffset, schemas.DataKind_int64, false),
	)

	viewSchema := smock.MockedSchema(testViewRecordQName1, schemas.SchemaKind_ViewRecord)
	viewSchema.MockContainers(
		smock.MockedContainer(schemas.SystemContainer_ViewPartitionKey, testViewRecordPkQName, 1, 1),
		smock.MockedContainer(schemas.SystemContainer_ViewValue, testViewRecordVQName, 1, 1),
	)

	cache := smock.MockedSchemaCache(
		viewSchema,
		pkSchema,
		valSchema,
	)

	appStructs := &mockAppStructs{}
	appStructs.
		On("ViewRecords").Return(viewRecords).
		On("Schemas").Return(cache).
		On("Events").Return(&nilEvents{}).
		On("Records").Return(&nilRecords{})
	return appStructs
}
func TestHostState_KeyBuilder_Should_return_unknown_storage_ID_error(t *testing.T) {
	require := require.New(t)
	s := hostStateForTest(&mockStorage{})

	_, err := s.KeyBuilder(schemas.NullQName, schemas.NullQName)

	require.ErrorIs(err, ErrUnknownStorage)
}
func TestHostState_CanExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k, err := s.KeyBuilder(testStorage, schemas.NullQName)
		require.Nil(err)

		_, ok, _ := s.CanExist(k)

		require.True(ok)
	})
	t.Run("Should return error when error occurred", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, err := s.KeyBuilder(testStorage, schemas.NullQName)
		require.Nil(err)

		_, _, err = s.CanExist(k)

		require.ErrorIs(err, errTest)
	})
	t.Run("Should return get batch not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, _ := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, _, err := s.CanExist(kb)

		require.ErrorIs(err, ErrGetBatchNotSupportedByStorage)
	})
}
func TestHostState_CanExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		times := 0
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k, err := s.KeyBuilder(testStorage, schemas.NullQName)
		require.Nil(err)

		_ = s.CanExistAll([]istructs.IStateKeyBuilder{k}, func(key istructs.IKeyBuilder, value istructs.IStateValue, ok bool) (err error) {
			times++
			require.Equal(k, key)
			require.True(ok)
			return
		})

		require.Equal(1, times)
	})
	t.Run("Should return error when error occurred", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.CanExistAll([]istructs.IStateKeyBuilder{k}, nil)

		require.ErrorIs(err, errTest)
	})
	t.Run("Should return get batch not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, _ := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.CanExistAll([]istructs.IStateKeyBuilder{kb}, nil)

		require.ErrorIs(err, ErrGetBatchNotSupportedByStorage)
	})
}
func TestHostState_MustExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := s.MustExist(k)

		require.NoError(err)
	})
	t.Run("Should return error when entity not exists", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = nil
			})
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := s.MustExist(k)

		require.ErrorIs(err, ErrNotExists)
	})
	t.Run("Should return error when error occurred on get batch", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := s.MustExist(k)

		require.ErrorIs(err, errTest)
	})
}
func TestHostState_MustExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
				args.Get(0).([]GetBatchItem)[1].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k1, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		k2, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		kk := make([]istructs.IKeyBuilder, 0, 2)

		_ = s.MustExistAll([]istructs.IStateKeyBuilder{k1, k2}, func(key istructs.IKeyBuilder, value istructs.IStateValue, ok bool) (err error) {
			kk = append(kk, key)
			require.True(ok)
			return
		})

		require.Equal(k1, kk[0])
		require.Equal(k1, kk[1])
	})
	t.Run("Should return error on get batch", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustExistAll([]istructs.IStateKeyBuilder{k}, nil)

		require.ErrorIs(err, errTest)
	})
	t.Run("Should return error when entity not exists", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
				args.Get(0).([]GetBatchItem)[1].value = nil
			})
		s := hostStateForTest(ms)
		k1, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		k2, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustExistAll([]istructs.IStateKeyBuilder{k1, k2}, nil)

		require.ErrorIs(err, ErrNotExists)
	})
	t.Run("Should return get batch not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, _ := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustExistAll([]istructs.IStateKeyBuilder{kb}, nil)

		require.ErrorIs(err, ErrGetBatchNotSupportedByStorage)
	})
}
func TestHostState_MustNotExist(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = nil
			})
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExist(k)

		require.NoError(err)
	})
	t.Run("Should return error when entity exists", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExist(k)

		require.ErrorIs(err, ErrExists)
	})
	t.Run("Should return error when error occurred on get batch", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExist(k)

		require.ErrorIs(err, errTest)
	})
}
func TestHostState_MustNotExistAll(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = nil
				args.Get(0).([]GetBatchItem)[1].value = nil
			})
		s := hostStateForTest(ms)
		k1, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		k2, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExistAll([]istructs.IStateKeyBuilder{k1, k2})

		require.NoError(err)
	})
	t.Run("Should return error on get batch", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).Return(errTest)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExistAll([]istructs.IStateKeyBuilder{k})

		require.ErrorIs(err, errTest)
	})
	t.Run("Should return error when entity exists", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("GetBatch", mock.AnythingOfType("[]state.GetBatchItem")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(0).([]GetBatchItem)[0].value = nil
				args.Get(0).([]GetBatchItem)[1].value = &mockStateValue{}
			})
		s := hostStateForTest(ms)
		k1, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		k2, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExistAll([]istructs.IStateKeyBuilder{k1, k2})

		require.ErrorIs(err, ErrExists)
	})
	t.Run("Should return get batch not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, _ := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.MustNotExistAll([]istructs.IStateKeyBuilder{kb})

		require.ErrorIs(err, ErrGetBatchNotSupportedByStorage)
	})
}
func TestHostState_Read(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("Read", mock.Anything, mock.AnythingOfType("istructs.ValueCallback")).Return(nil)
		s := hostStateForTest(ms)
		k, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_ = s.Read(k, nil)

		ms.AssertExpectations(t)
	})
	t.Run("Should return read not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, _ := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		err := s.Read(kb, nil)

		require.ErrorIs(err, ErrReadNotSupportedByStorage)
	})
}
func TestHostState_NewValue(t *testing.T) {
	t.Run("Should return error when intents limit exceeded", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, i := limitedIntentsHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := i.NewValue(kb)

		require.ErrorIs(err, ErrIntentsLimitExceeded)
	})
	t.Run("Should return insert not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, i := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := i.NewValue(kb)

		require.ErrorIs(err, ErrInsertNotSupportedByStorage)
	})
}
func TestHostState_UpdateValue(t *testing.T) {
	t.Run("Should return error when intents limit exceeded", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, i := limitedIntentsHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := i.UpdateValue(kb, nil)

		require.ErrorIs(err, ErrIntentsLimitExceeded)
	})
	t.Run("Should return update not supported by storage error", func(t *testing.T) {
		require := require.New(t)
		ms := &mockStorage{}
		ms.On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName))
		s, i := emptyHostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)

		_, err := i.UpdateValue(kb, nil)

		require.ErrorIs(err, ErrUpdateNotSupportedByStorage)
	})
}
func TestHostState_ValidateIntents(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("ProvideValueBuilder", mock.Anything, mock.Anything).Return(&viewRecordsValueBuilder{}).
			On("Validate", mock.Anything).Return(nil)
		s := hostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		_, _ = s.NewValue(kb)

		err := s.ValidateIntents()

		require.NoError(t, err)
	})
	t.Run("Should return immediately when intents are empty", func(t *testing.T) {
		ms := &mockStorage{}
		s := hostStateForTest(&mockStorage{})

		_ = s.ValidateIntents()

		ms.AssertNotCalled(t, "Validate", mock.Anything)
	})
	t.Run("Should return validation error", func(t *testing.T) {
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("ProvideValueBuilder", mock.Anything, mock.Anything).Return(&viewRecordsValueBuilder{}).
			On("Validate", mock.Anything).Return(errTest)
		s := hostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		_, _ = s.NewValue(kb)

		err := s.ValidateIntents()

		require.ErrorIs(t, err, errTest)
	})
}
func TestHostState_ApplyIntents(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("ProvideValueBuilder", mock.Anything, mock.Anything).Return(&viewRecordsValueBuilder{}).
			On("ApplyBatch", mock.Anything).Return(nil)
		s := hostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		_, _ = s.NewValue(kb)

		_ = s.ApplyIntents()

		ms.AssertExpectations(t)
	})
	t.Run("Should return apply batch error", func(t *testing.T) {
		ms := &mockStorage{}
		ms.
			On("NewKeyBuilder", schemas.NullQName, nil).Return(newKeyBuilder(testStorage, schemas.NullQName)).
			On("ProvideValueBuilder", mock.Anything, mock.Anything).Return(&viewRecordsValueBuilder{}).
			On("ApplyBatch", mock.Anything).Return(errTest)
		s := hostStateForTest(ms)
		kb, _ := s.KeyBuilder(testStorage, schemas.NullQName)
		_, _ = s.NewValue(kb)

		err := s.ApplyIntents()

		require.ErrorIs(t, err, errTest)
	})
}
func hostStateForTest(s IStateStorage) IHostState {
	hs := newHostState("ForTest", 10)
	hs.addStorage(testStorage, s, S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	return hs
}
func emptyHostStateForTest(s IStateStorage) (istructs.IState, istructs.IIntents) {
	bs := ProvideQueryProcessorStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, nil).(*hostState)
	bs.addStorage(testStorage, s, math.MinInt)
	return bs, bs
}
func limitedIntentsHostStateForTest(s IStateStorage) (istructs.IState, istructs.IIntents) {
	hs := newHostState("LimitedIntentsForTest", 0)
	hs.addStorage(testStorage, s, S_GET_BATCH|S_READ|S_INSERT|S_UPDATE)
	return hs, hs
}
