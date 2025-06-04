/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package recreg_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/recreg"
)

type mockViewRecords struct {
	mock.Mock
	istructs.IViewRecords
}

func (v *mockViewRecords) KeyBuilder(name appdef.QName) istructs.IKeyBuilder {
	return v.Called(name).Get(0).(istructs.IKeyBuilder)
}

func (v *mockViewRecords) Get(ws istructs.WSID, key istructs.IKeyBuilder) (value istructs.IValue, err error) {
	called := v.Called(ws, key)
	return called.Get(0).(istructs.IValue), called.Error(1)
}

type mockKey struct {
	mock.Mock
	istructs.IKeyBuilder
}

func (k *mockKey) PutInt64(n appdef.FieldName, v int64)                { k.Called(n, v) }
func (k *mockKey) PutRecordID(n appdef.FieldName, v istructs.RecordID) { k.Called(n, v) }

type mockValue struct {
	mock.Mock
	istructs.IValue
}

func (v *mockValue) AsInt64(n appdef.FieldName) int64 { return v.Called(n).Get(0).(int64) }
func (v *mockValue) AsQName(n appdef.FieldName) appdef.QName {
	return v.Called(n).Get(0).(appdef.QName)
}

func Test_BasicUsage(t *testing.T) {
	mockView := &mockViewRecords{}
	mockKeyBuilder := &mockKey{}
	mockValue := &mockValue{}

	registry := recreg.New(func() istructs.IViewRecords { return mockView })

	wsid := istructs.WSID(100)
	id := istructs.RecordID(12345)
	expectedQName := appdef.NewQName("test", "TestRecord")
	expectedOffset := istructs.Offset(67890)

	mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
	mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
	mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id).Once()
	mockView.On("Get", wsid, mockKeyBuilder).Return(mockValue, nil).Once()
	mockValue.On("AsQName", sys.RecordsRegistryView.Fields.QName).Return(expectedQName).Once()
	mockValue.On("AsInt64", sys.RecordsRegistryView.Fields.WLogOffset).Return(int64(expectedOffset)).Once()

	qName, offset, err := registry.Get(wsid, id)

	require := require.New(t)

	require.NoError(err)
	require.Equal(expectedQName, qName)
	require.Equal(expectedOffset, offset)

	mockView.AssertExpectations(t)
	mockKeyBuilder.AssertExpectations(t)
	mockValue.AssertExpectations(t)
}

func Test_Errors(t *testing.T) {
	wsid := istructs.WSID(100)
	id := istructs.RecordID(12345)

	t.Run("Should return nul if not found", func(t *testing.T) {
		mockView := &mockViewRecords{}
		mockKeyBuilder := &mockKey{}
		mockValue := &mockValue{}

		mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
		mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
		mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id)
		mockView.On("Get", wsid, mockKeyBuilder).Return(mockValue, istructs.ErrRecordNotFound).Once()

		registry := recreg.New(func() istructs.IViewRecords { return mockView })
		n, o, err := registry.Get(wsid, id)

		require := require.New(t)

		require.NoError(err)
		require.Equal(appdef.NullQName, n)
		require.Equal(istructs.NullOffset, o)

		mockView.AssertExpectations(t)
		mockKeyBuilder.AssertExpectations(t)
	})

	t.Run("Should return nul and error if registry get failed", func(t *testing.T) {
		testError := errors.New("test error")

		mockView := &mockViewRecords{}
		mockKeyBuilder := &mockKey{}
		mockValue := &mockValue{}

		mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
		mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
		mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id)
		mockView.On("Get", wsid, mockKeyBuilder).Return(mockValue, testError).Once()

		registry := recreg.New(func() istructs.IViewRecords { return mockView })
		n, o, err := registry.Get(wsid, id)

		require := require.New(t)

		require.Error(err, require.Is(testError), require.HasAll(wsid, id))
		require.Equal(appdef.NullQName, n)
		require.Equal(istructs.NullOffset, o)

		mockView.AssertExpectations(t)
		mockKeyBuilder.AssertExpectations(t)
	})
}
