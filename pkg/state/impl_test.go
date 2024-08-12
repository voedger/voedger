/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

var (
	testRecordQName1      = appdef.NewQName("test", "record1")
	testRecordQName2      = appdef.NewQName("test", "record2")
	testRecordQName3      = appdef.NewQName("test", "record3")
	testViewRecordQName1  = appdef.NewQName("test", "viewRecord1")
	testViewRecordQName2  = appdef.NewQName("test", "viewRecord2")
	testStorage           = appdef.NewQName("test", "testStorage")
	testWSQName           = appdef.NewQName("test", "testWS")
	testWSDescriptorQName = appdef.NewQName("test", "testWSDescriptor")
	testAppQName          = appdef.NewAppQName("test", "testApp")
)

func TestSimpleWSIDFunc(t *testing.T) {
	require.Equal(t, istructs.WSID(10), SimpleWSIDFunc(istructs.WSID(10))())
}
func TestSimplePartitionIDFuncDFunc(t *testing.T) {
	require.Equal(t, istructs.PartitionID(10), SimplePartitionIDFunc(istructs.PartitionID(10))())
}
func Test_put(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		mrw := &mockRowWriter{}
		mrw.
			On("PutInt32", "int32Fld", int32(1)).
			On("PutInt64", "int64Fld", int64(2)).
			On("PutFloat32", "float32Fld", float32(3.1)).
			On("PutFloat64", "float64Fld", 4.2).
			On("PutBytes", "byteFld", []byte{5}).
			On("PutString", "stringFld", "string").
			On("PutQName", "qNameFld", testRecordQName1).
			On("PutBool", "boolFld", true).
			On("PutRecordID", "recordIDFld", istructs.RecordID(6))
		mrr := &mockRowReader{}
		mrr.
			On("AsInt32", "int32Fld").Return(int32(1)).
			On("AsInt64", "int64Fld").Return(int64(2)).
			On("AsFloat32", "float32Fld").Return(float32(3.1)).
			On("AsFloat64", "float64Fld").Return(4.2).
			On("AsBytes", "byteFld").Return([]byte{5}).
			On("AsString", "stringFld").Return("string").
			On("AsQName", "qNameFld").Return(testRecordQName1).
			On("AsBool", "boolFld").Return(true).
			On("AsRecordID", "recordIDFld").Return(istructs.RecordID(6))

		put("int32Fld", appdef.DataKind_int32, mrr, mrw)
		put("int64Fld", appdef.DataKind_int64, mrr, mrw)
		put("float32Fld", appdef.DataKind_float32, mrr, mrw)
		put("float64Fld", appdef.DataKind_float64, mrr, mrw)
		put("byteFld", appdef.DataKind_bytes, mrr, mrw)
		put("stringFld", appdef.DataKind_string, mrr, mrw)
		put("qNameFld", appdef.DataKind_QName, mrr, mrw)
		put("boolFld", appdef.DataKind_bool, mrr, mrw)
		put("recordIDFld", appdef.DataKind_RecordID, mrr, mrw)

		mrw.AssertExpectations(t)
		mrr.AssertExpectations(t)
	})
	t.Run("Should panic when data kind not supported", func(t *testing.T) {
		require.PanicsWithError(t,
			fmt.Sprintf("illegal state: field - 'notSupported', kind - '%d': not supported", appdef.DataKind_FakeLast),
			func() { put("notSupported", appdef.DataKind_FakeLast, nil, nil) })
	})
}

func Test_getStorageID(t *testing.T) {
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
				kb:              &mailKeyBuilder{},
				expectedStorage: sys.Storage_SendMail,
			},
			{
				name:            "HTTP storage key",
				kb:              &httpStorageKeyBuilder{},
				expectedStorage: sys.Storage_Http,
			},
			{
				name:            "View storage key",
				kb:              &viewKeyBuilder{},
				expectedStorage: sys.Storage_View,
			},
		}
		for _, test := range tests {
			require.Equal(t, test.expectedStorage, test.kb.Storage())
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

type nilRecords struct {
	istructs.IRecords
}

type nilAppDef struct {
	appdef.IAppDef
}

type nilViewRecords struct {
	istructs.IViewRecords
}

type mockAppStructs struct {
	istructs.IAppStructs
	mock.Mock
}

func (s *mockAppStructs) AppQName() appdef.AppQName {
	return s.Called().Get(0).(appdef.AppQName)
}
func (s *mockAppStructs) AppDef() appdef.IAppDef {
	return s.Called().Get(0).(appdef.IAppDef)
}
func (s *mockAppStructs) Events() istructs.IEvents   { return s.Called().Get(0).(istructs.IEvents) }
func (s *mockAppStructs) Records() istructs.IRecords { return s.Called().Get(0).(istructs.IRecords) }
func (s *mockAppStructs) ViewRecords() istructs.IViewRecords {
	return s.Called().Get(0).(istructs.IViewRecords)
}

type mockEvents struct {
	istructs.IEvents
	mock.Mock
}

func (e *mockEvents) ReadPLog(ctx context.Context, partition istructs.PartitionID, offset istructs.Offset, toReadCount int, cb istructs.PLogEventsReaderCallback) (err error) {
	return e.Called(ctx, partition, offset, toReadCount, cb).Error(0)
}
func (e *mockEvents) ReadWLog(ctx context.Context, workspace istructs.WSID, offset istructs.Offset, toReadCount int, cb istructs.WLogEventsReaderCallback) (err error) {
	return e.Called(ctx, workspace, offset, toReadCount, cb).Error(0)
}

type mockRecords struct {
	istructs.IRecords
	mock.Mock
}

func (r *mockRecords) GetBatch(workspace istructs.WSID, highConsistency bool, ids []istructs.RecordGetBatchItem) (err error) {
	return r.Called(workspace, highConsistency, ids).Error(0)
}
func (r *mockRecords) Get(workspace istructs.WSID, highConsistency bool, id istructs.RecordID) (record istructs.IRecord, err error) {
	args := r.Called(workspace, highConsistency, id)

	if args.Get(0) == nil {
		return nil, args.Error(1)

	}
	return args.Get(0).(istructs.IRecord), args.Error(1)
}
func (r *mockRecords) GetSingleton(workspace istructs.WSID, qName appdef.QName) (record istructs.IRecord, err error) {
	aa := r.Called(workspace, qName)
	return aa.Get(0).(istructs.IRecord), aa.Error(1)
}

type mockViewRecords struct {
	istructs.IViewRecords
	mock.Mock
}

func (r *mockViewRecords) KeyBuilder(view appdef.QName) istructs.IKeyBuilder {
	return r.Called(view).Get(0).(istructs.IKeyBuilder)
}
func (r *mockViewRecords) NewValueBuilder(view appdef.QName) istructs.IValueBuilder {
	return r.Called(view).Get(0).(istructs.IValueBuilder)
}
func (r *mockViewRecords) UpdateValueBuilder(view appdef.QName, existing istructs.IValue) istructs.IValueBuilder {
	return r.Called(view, existing).Get(0).(istructs.IValueBuilder)
}
func (r *mockViewRecords) Get(workspace istructs.WSID, key istructs.IKeyBuilder) (value istructs.IValue, err error) {
	c := r.Called(workspace, key)
	if c.Get(0) == nil {
		return nil, c.Error(1)
	}
	return c.Get(0).(istructs.IValue), c.Error(1)
}
func (r *mockViewRecords) GetBatch(workspace istructs.WSID, kv []istructs.ViewRecordGetBatchItem) (err error) {
	return r.Called(workspace, kv).Error(0)
}
func (r *mockViewRecords) PutBatch(workspace istructs.WSID, batch []istructs.ViewKV) (err error) {
	return r.Called(workspace, batch).Error(0)
}
func (r *mockViewRecords) Read(ctx context.Context, workspace istructs.WSID, key istructs.IKeyBuilder, cb istructs.ValuesCallback) (err error) {
	return r.Called(ctx, workspace, key, cb).Error(0)
}

type mockRecord struct {
	istructs.IRecord
	mock.Mock
}

func (r *mockRecord) QName() appdef.QName       { return r.Called().Get(0).(appdef.QName) }
func (r *mockRecord) AsInt64(name string) int64 { return r.Called(name).Get(0).(int64) }
func (r *mockRecord) AsQName(name string) appdef.QName {
	return r.Called(name).Get(0).(appdef.QName)
}
func (r *mockRecord) FieldNames(cb func(fieldName string)) {
	r.Called(cb)
}

type mockValue struct {
	istructs.IValue
	mock.Mock
}

func (v *mockValue) AsInt32(name string) int32     { return v.Called(name).Get(0).(int32) }
func (v *mockValue) AsInt64(name string) int64     { return v.Called(name).Get(0).(int64) }
func (v *mockValue) AsFloat32(name string) float32 { return v.Called(name).Get(0).(float32) }
func (v *mockValue) AsFloat64(name string) float64 { return v.Called(name).Get(0).(float64) }
func (v *mockValue) AsBytes(name string) []byte    { return v.Called(name).Get(0).([]byte) }
func (v *mockValue) AsString(name string) string   { return v.Called(name).String(0) }
func (v *mockValue) AsQName(name string) appdef.QName {
	return v.Called(name).Get(0).(appdef.QName)
}
func (v *mockValue) AsBool(name string) bool { return v.Called(name).Bool(0) }
func (v *mockValue) AsRecordID(name string) istructs.RecordID {
	return v.Called(name).Get(0).(istructs.RecordID)
}
func (v *mockValue) AsRecord(name string) (record istructs.IRecord) {
	return v.Called(name).Get(0).(istructs.IRecord)
}
func (v *mockValue) AsEvent(name string) (event istructs.IDbEvent) {
	return v.Called(name).Get(0).(istructs.IDbEvent)
}

type nilKey struct {
	istructs.IKey
}

type nilValue struct {
	istructs.IValue
}

type mockStorage struct {
	IWithInsert
	mock.Mock
}

func (s *mockStorage) NewKeyBuilder(entity appdef.QName, existingBuilder istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return s.Called(entity, existingBuilder).Get(0).(istructs.IStateKeyBuilder)
}
func (s *mockStorage) GetBatch(items []GetBatchItem) (err error) {
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
func (s *mockStorage) Validate(items []ApplyBatchItem) (err error) {
	return s.Called(items).Error(0)
}
func (s *mockStorage) ApplyBatch(items []ApplyBatchItem) (err error) {
	return s.Called(items).Error(0)
}

type mockKeyBuilder struct {
	istructs.IKeyBuilder
	mock.Mock
}

func (b *mockKeyBuilder) PutInt64(name string, value int64)                { b.Called(name, value) }
func (b *mockKeyBuilder) PutString(name, value string)                     { b.Called(name, value) }
func (b *mockKeyBuilder) PutRecordID(name string, value istructs.RecordID) { b.Called(name, value) }
func (b *mockKeyBuilder) PutQName(name string, value appdef.QName)         { b.Called(name, value) }
func (b *mockKeyBuilder) Equals(src istructs.IKeyBuilder) bool             { return b.Called(src).Bool(0) }

type nilKeyBuilder struct {
	istructs.IKeyBuilder
}

func (b *nilKeyBuilder) Equals(istructs.IKeyBuilder) bool { return false }

type mockValueBuilder struct {
	istructs.IValueBuilder
	mock.Mock
}

func (b *mockValueBuilder) PutInt32(name string, value int32)                { b.Called(name, value) }
func (b *mockValueBuilder) PutInt64(name string, value int64)                { b.Called(name, value) }
func (b *mockValueBuilder) PutFloat32(name string, value float32)            { b.Called(name, value) }
func (b *mockValueBuilder) PutFloat64(name string, value float64)            { b.Called(name, value) }
func (b *mockValueBuilder) PutBytes(name string, value []byte)               { b.Called(name, value) }
func (b *mockValueBuilder) PutString(name string, value string)              { b.Called(name, value) }
func (b *mockValueBuilder) PutQName(name string, value appdef.QName)         { b.Called(name, value) }
func (b *mockValueBuilder) PutBool(name string, value bool)                  { b.Called(name, value) }
func (b *mockValueBuilder) PutRecordID(name string, value istructs.RecordID) { b.Called(name, value) }
func (b *mockValueBuilder) Build() istructs.IValue                           { return b.Called().Get(0).(istructs.IValue) }

type nilValueBuilder struct {
	istructs.IValueBuilder
}

type mockRowReader struct {
	istructs.IRowReader
	mock.Mock
}

func (r *mockRowReader) AsInt32(name string) int32     { return r.Called(name).Get(0).(int32) }
func (r *mockRowReader) AsInt64(name string) int64     { return r.Called(name).Get(0).(int64) }
func (r *mockRowReader) AsFloat32(name string) float32 { return r.Called(name).Get(0).(float32) }
func (r *mockRowReader) AsFloat64(name string) float64 { return r.Called(name).Get(0).(float64) }
func (r *mockRowReader) AsBytes(name string) []byte    { return r.Called(name).Get(0).([]byte) }
func (r *mockRowReader) AsString(name string) string   { return r.Called(name).String(0) }
func (r *mockRowReader) AsQName(name string) appdef.QName {
	return r.Called(name).Get(0).(appdef.QName)
}
func (r *mockRowReader) AsBool(name string) bool { return r.Called(name).Bool(0) }
func (r *mockRowReader) AsRecordID(name string) istructs.RecordID {
	return r.Called(name).Get(0).(istructs.RecordID)
}

type mockRowWriter struct {
	istructs.IRowWriter
	mock.Mock
}

func (w *mockRowWriter) PutInt32(name string, value int32)                { w.Called(name, value) }
func (w *mockRowWriter) PutInt64(name string, value int64)                { w.Called(name, value) }
func (w *mockRowWriter) PutFloat32(name string, value float32)            { w.Called(name, value) }
func (w *mockRowWriter) PutFloat64(name string, value float64)            { w.Called(name, value) }
func (w *mockRowWriter) PutBytes(name string, value []byte)               { w.Called(name, value) }
func (w *mockRowWriter) PutString(name, value string)                     { w.Called(name, value) }
func (w *mockRowWriter) PutQName(name string, value appdef.QName)         { w.Called(name, value) }
func (w *mockRowWriter) PutBool(name string, value bool)                  { w.Called(name, value) }
func (w *mockRowWriter) PutRecordID(name string, value istructs.RecordID) { w.Called(name, value) }

type mockWLogEvent struct {
	mock.Mock
}

func (e *mockWLogEvent) ArgumentObject() istructs.IObject {
	return e.Called().Get(0).(istructs.IObject)
}
func (e *mockWLogEvent) Bytes() []byte { return e.Called().Get(0).([]byte) }
func (e *mockWLogEvent) CUDs(cb func(rec istructs.ICUDRow)) {
	e.Called(cb)
}
func (e *mockWLogEvent) RegisteredAt() istructs.UnixMilli {
	return e.Called().Get(0).(istructs.UnixMilli)
}
func (e *mockWLogEvent) DeviceID() istructs.ConnectedDeviceID {
	return e.Called().Get(0).(istructs.ConnectedDeviceID)
}
func (e *mockWLogEvent) Synced() bool                 { return e.Called().Bool(0) }
func (e *mockWLogEvent) QName() appdef.QName          { return e.Called().Get(0).(appdef.QName) }
func (e *mockWLogEvent) SyncedAt() istructs.UnixMilli { return e.Called().Get(0).(istructs.UnixMilli) }
func (e *mockWLogEvent) Error() istructs.IEventError  { return e.Called().Get(0).(istructs.IEventError) }
func (e *mockWLogEvent) Release()                     {}

type mockCUD struct {
	mock.Mock
}

func (c *mockCUD) Create(qName appdef.QName) istructs.IRowWriter {
	return c.Called().Get(0).(istructs.IRowWriter)
}
func (c *mockCUD) Update(record istructs.IRecord) istructs.IRowWriter {
	return c.Called(record).Get(0).(istructs.IRowWriter)
}

type mockStateValue struct {
	istructs.IStateValue
	mock.Mock
}

type mockCUDRow struct {
	istructs.ICUDRow
	mock.Mock
}
