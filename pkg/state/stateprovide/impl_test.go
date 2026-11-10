/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package stateprovide

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
)

var (
	testRecordQName1      = appdef.NewQName("test", "record1")
	testRecordQName2      = appdef.NewQName("test", "record2")
	testRecordQName3      = appdef.NewQName("test", "record3")
	testViewRecordQName1  = appdef.NewQName("test", "viewRecord1")
	testViewRecordQName2  = appdef.NewQName("test", "viewRecord2")
	testStorage           = appdef.NewQName("test", "testStorage")
	testEntity            = appdef.NewQName("test", "testEntity")
	testWSQName           = appdef.NewQName("test", "testWS")
	testWSDescriptorQName = appdef.NewQName("test", "testWSDescriptor")
	testAppQName          = appdef.NewAppQName("test", "testApp")
)

func TestSimpleWSIDFunc(t *testing.T) {
	require := require.New(t)
	require.Equal(istructs.WSID(10), state.SimpleWSIDFunc(istructs.WSID(10))())
}
func TestSimplePartitionIDFuncDFunc(t *testing.T) {
	require := require.New(t)
	require.Equal(istructs.PartitionID(10), state.SimplePartitionIDFunc(istructs.PartitionID(10))())
}

func Test_getStorageID(t *testing.T) {
	require := require.New(t)
	t.Run("Should be ok", func(t *testing.T) {
		tests := []struct {
			name            string
			kb              istructs.IStateKeyBuilder
			expectedStorage appdef.QName
		}{
			{
				name:            "General storage key",
				kb:              newMapKeyBuilder(sys.Storage_Record, appdef.NullQName),
				expectedStorage: sys.Storage_Record,
			},
			{
				name:            "Email storage key",
				kb:              newMapKeyBuilder(sys.Storage_SendMail, appdef.NullQName),
				expectedStorage: sys.Storage_SendMail,
			},
		}
		for _, test := range tests {
			require.Equal(test.expectedStorage, test.kb.Storage())
		}
	})
}

type nilAppStructs struct {
	istructs.IAppStructs
}

func nilAppStructsFunc() istructs.IAppStructs {
	return &nilAppStructs{}
}

func (s *nilAppStructs) AppDef() appdef.IAppDef             { return nil }
func (s *nilAppStructs) Events() istructs.IEvents           { return nil }
func (s *nilAppStructs) Records() istructs.IRecords         { return nil }
func (s *nilAppStructs) ViewRecords() istructs.IViewRecords { return nil }

type nilEvents struct {
	istructs.IEvents
}

type nilKey struct {
	istructs.IKey
}

type nilValue struct {
	istructs.IValue
}

type mockStorage struct {
	state.IWithInsert
	mock.Mock
}

func (s *mockStorage) NewKeyBuilder(entity appdef.QName, existingBuilder istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return s.Called(entity, existingBuilder).Get(0).(istructs.IStateKeyBuilder)
}
func (s *mockStorage) GetBatch(items []state.GetBatchItem) (err error) {
	return s.Called(items).Error(0)
}
func (s *mockStorage) Get(key istructs.IStateKeyBuilder) (value istructs.IStateValue, err error) {
	return s.Called(key).Get(0).(istructs.IStateValue), s.Called(key).Error(1)
}
func (s *mockStorage) Read(key istructs.IStateKeyBuilder, callback istructs.ValueCallback) (err error) {
	return s.Called(key, callback).Error(0)
}
func (s *mockStorage) ProvideValueBuilder(key istructs.IStateKeyBuilder, existingBuilder istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return s.Called(key, existingBuilder).Get(0).(istructs.IStateValueBuilder), nil
}
func (s *mockStorage) ProvideValueBuilderForUpdate(key istructs.IStateKeyBuilder, existingValue istructs.IStateValue, existingBuilder istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return s.Called(key, existingValue, existingBuilder).Get(0).(istructs.IStateValueBuilder), nil
}
func (s *mockStorage) Validate(items []state.ApplyBatchItem) (err error) {
	return s.Called(items).Error(0)
}
func (s *mockStorage) ApplyBatch(items []state.ApplyBatchItem) (err error) {
	return s.Called(items).Error(0)
}

type nilKeyBuilder struct {
	istructs.IKeyBuilder
}

func (b *nilKeyBuilder) Equals(istructs.IKeyBuilder) bool { return false }

type nilValueBuilder struct {
	istructs.IValueBuilder
}
