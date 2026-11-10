/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package recreg_test

import (
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/recreg"
)

func TestRegistryReturnsRecordTypeAndOffset(t *testing.T) {
	require := require.New(t)
	mockView := &coreutils.MockViewRecords{}
	mockKeyBuilder := &coreutils.MockStateKeyBuilder{}
	value := &coreutils.MockStateValue{}

	registry := recreg.New(func() istructs.IViewRecords { return mockView })

	wsid := istructs.WSID(100)
	id := istructs.RecordID(12345)
	expectedQName := appdef.NewQName("test", "TestRecord")
	expectedOffset := istructs.Offset(67890)

	mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
	mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
	mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id).Once()
	mockView.On("Get", wsid, mockKeyBuilder).Return(value, nil).Once()
	value.On("AsQName", sys.RecordsRegistryView.Fields.QName).Return(expectedQName).Once()
	value.On("AsInt64", sys.RecordsRegistryView.Fields.WLogOffset).Return(int64(expectedOffset)).Once()

	// Return the stored record type and offset
	actualQName, actualOffset, err := registry.Get(wsid, id)

	require.NoError(err)
	require.Equal(expectedQName, actualQName)
	require.Equal(expectedOffset, actualOffset)

	mockView.AssertExpectations(t)
	mockKeyBuilder.AssertExpectations(t)
	value.AssertExpectations(t)
}

func TestRegistryLookupErrors(t *testing.T) {
	require := require.New(t)
	wsid := istructs.WSID(100)
	id := istructs.RecordID(12345)

	t.Run("Missing record returns no type or offset", func(t *testing.T) {
		mockView := &coreutils.MockViewRecords{}
		mockKeyBuilder := &coreutils.MockStateKeyBuilder{}
		value := &coreutils.MockStateValue{}

		mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
		mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
		mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id)
		mockView.On("Get", wsid, mockKeyBuilder).Return(value, istructs.ErrRecordNotFound).Once()

		registry := recreg.New(func() istructs.IViewRecords { return mockView })
		// Preserve missing-record handling and storage error context
		actualQName, actualOffset, err := registry.Get(wsid, id)

		require.NoError(err)
		require.Equal(appdef.NullQName, actualQName)
		require.Equal(istructs.NullOffset, actualOffset)

		mockView.AssertExpectations(t)
		mockKeyBuilder.AssertExpectations(t)
	})

	t.Run("Storage failure preserves its cause and lookup identifiers", func(t *testing.T) {
		testError := errors.New("test error")

		mockView := &coreutils.MockViewRecords{}
		mockKeyBuilder := &coreutils.MockStateKeyBuilder{}
		value := &coreutils.MockStateValue{}

		mockView.On("KeyBuilder", sys.RecordsRegistryView.Name).Return(mockKeyBuilder).Once()
		mockKeyBuilder.On("PutInt64", sys.RecordsRegistryView.Fields.IDHi, sys.RecordsRegistryView.Fields.CrackID(id)).Once()
		mockKeyBuilder.On("PutRecordID", sys.RecordsRegistryView.Fields.ID, id)
		mockView.On("Get", wsid, mockKeyBuilder).Return(value, testError).Once()

		registry := recreg.New(func() istructs.IViewRecords { return mockView })
		// Preserve missing-record handling and storage error context
		actualQName, actualOffset, err := registry.Get(wsid, id)

		require.Error(err, require.Is(testError), require.HasAll(wsid, id))
		require.Equal(appdef.NullQName, actualQName)
		require.Equal(istructs.NullOffset, actualOffset)

		mockView.AssertExpectations(t)
		mockKeyBuilder.AssertExpectations(t)
	})
}
