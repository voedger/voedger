/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

var (
	testRecordQName1      = appdef.NewQName("test", "record1")
	testRecordQName2      = appdef.NewQName("test", "record2")
	testViewRecordQName1  = appdef.NewQName("test", "viewRecord1")
	testViewRecordQName2  = appdef.NewQName("test", "viewRecord2")
	testWSQName           = appdef.NewQName("test", "testWS")
	testWSDescriptorQName = appdef.NewQName("test", "testWSDescriptor")
	testAppQName          = appdef.NewAppQName("test", "testApp")
)

type nilEvents struct {
	istructs.IEvents
}

type mockCUD struct {
	mock.Mock
}

func (c *mockCUD) Create(qName appdef.QName) istructs.IRowWriter {
	return c.Called().Get(0).(istructs.IRowWriter)
}
func (c *mockCUD) Update(record istructs.IRecord) istructs.IRowWriter {
	return c.Called(record).Get(0).(istructs.IRowWriter)
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

type nilViewRecords struct {
	istructs.IViewRecords
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
func (r *mockRecord) FieldNames(cb func(string) bool) {
	r.Called(cb)
}

func put(fieldName string, kind appdef.DataKind, rr istructs.IRowReader, rw istructs.IRowWriter) {
	switch kind {
	case appdef.DataKind_int8:
		rw.PutInt8(fieldName, rr.AsInt8(fieldName))
	case appdef.DataKind_int16:
		rw.PutInt16(fieldName, rr.AsInt16(fieldName))
	case appdef.DataKind_int32:
		rw.PutInt32(fieldName, rr.AsInt32(fieldName))
	case appdef.DataKind_int64:
		rw.PutInt64(fieldName, rr.AsInt64(fieldName))
	case appdef.DataKind_float32:
		rw.PutFloat32(fieldName, rr.AsFloat32(fieldName))
	case appdef.DataKind_float64:
		rw.PutFloat64(fieldName, rr.AsFloat64(fieldName))
	case appdef.DataKind_bytes:
		rw.PutBytes(fieldName, rr.AsBytes(fieldName))
	case appdef.DataKind_string:
		rw.PutString(fieldName, rr.AsString(fieldName))
	case appdef.DataKind_QName:
		rw.PutQName(fieldName, rr.AsQName(fieldName))
	case appdef.DataKind_bool:
		rw.PutBool(fieldName, rr.AsBool(fieldName))
	case appdef.DataKind_RecordID:
		rw.PutRecordID(fieldName, rr.AsRecordID(fieldName))
	default:
		panic(fmt.Errorf("illegal state: field - '%s', kind - '%d': %w", fieldName, kind, ErrNotSupported))
	}
}

func Test_put(t *testing.T) {
	t.Run("Should be ok", func(t *testing.T) {
		mrw := &mockRowWriter{}
		mrw.
			On("PutInt8", "int8Fld", int8(-2)).
			On("PutInt16", "int16Fld", int16(-1)).
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
			On("AsInt8", "int8Fld").Return(int8(-2)).
			On("AsInt16", "int16Fld").Return(int16(-1)).
			On("AsInt32", "int32Fld").Return(int32(1)).
			On("AsInt64", "int64Fld").Return(int64(2)).
			On("AsFloat32", "float32Fld").Return(float32(3.1)).
			On("AsFloat64", "float64Fld").Return(4.2).
			On("AsBytes", "byteFld").Return([]byte{5}).
			On("AsString", "stringFld").Return("string").
			On("AsQName", "qNameFld").Return(testRecordQName1).
			On("AsBool", "boolFld").Return(true).
			On("AsRecordID", "recordIDFld").Return(istructs.RecordID(6))

		put("int8Fld", appdef.DataKind_int8, mrr, mrw)
		put("int16Fld", appdef.DataKind_int16, mrr, mrw)
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

type mockRowReader struct {
	istructs.IRowReader
	mock.Mock
}

func (r *mockRowReader) AsInt8(name string) int8       { return r.Called(name).Get(0).(int8) }
func (r *mockRowReader) AsInt16(name string) int16     { return r.Called(name).Get(0).(int16) }
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

func (w *mockRowWriter) PutInt8(name string, value int8)                  { w.Called(name, value) }
func (w *mockRowWriter) PutInt16(name string, value int16)                { w.Called(name, value) }
func (w *mockRowWriter) PutInt32(name string, value int32)                { w.Called(name, value) }
func (w *mockRowWriter) PutInt64(name string, value int64)                { w.Called(name, value) }
func (w *mockRowWriter) PutFloat32(name string, value float32)            { w.Called(name, value) }
func (w *mockRowWriter) PutFloat64(name string, value float64)            { w.Called(name, value) }
func (w *mockRowWriter) PutBytes(name string, value []byte)               { w.Called(name, value) }
func (w *mockRowWriter) PutString(name, value string)                     { w.Called(name, value) }
func (w *mockRowWriter) PutQName(name string, value appdef.QName)         { w.Called(name, value) }
func (w *mockRowWriter) PutBool(name string, value bool)                  { w.Called(name, value) }
func (w *mockRowWriter) PutRecordID(name string, value istructs.RecordID) { w.Called(name, value) }

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
func mockedStructs(t *testing.T) (*mockAppStructs, *mockViewRecords) {
	appDef := builder.New()

	appDef.AddPackage("test", "test.com/test")

	wsb := appDef.AddWorkspace(testWSQName)
	wsDesc := wsb.AddCDoc(testWSDescriptorQName)
	wsDesc.AddField(field_WSKind, appdef.DataKind_bytes, false)
	wsb.SetDescriptor(testWSDescriptorQName)

	view := wsb.AddView(testViewRecordQName1)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)
	view.Value().AddField("i64", appdef.DataKind_int64, false)
	view.Value().AddField("recID", appdef.DataKind_RecordID, false)

	view = wsb.AddView(testViewRecordQName2)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	app, err := appDef.Build()
	require.NoError(t, err)

	mockWorkspaceRecord := &mockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)
	mockedRecords := &mockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	mockedViews := &mockViewRecords{}
	mockedViews.On("KeyBuilder", testViewRecordQName1).Return(&viewKeyBuilder{IKeyBuilder: newUniqKeyBuilder(sys.Storage_View, appdef.NullQName), view: testViewRecordQName1})

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(app).
		On("AppQName").Return(testAppQName).
		On("Records").Return(mockedRecords).
		On("Events").Return(&nilEvents{}).
		On("ViewRecords").Return(mockedViews)

	return appStructs, mockedViews
}

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
type nilValue struct {
	istructs.IValue
}

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
type nilKeyBuilder struct {
	istructs.IKeyBuilder
}

func (b *nilKeyBuilder) Equals(istructs.IKeyBuilder) bool { return false }

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
type nilValueBuilder struct {
	istructs.IValueBuilder
}

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
type mockValue struct {
	istructs.IValue
	mock.Mock
}

func (v *mockValue) AsInt8(name string) int8       { return v.Called(name).Get(0).(int8) }
func (v *mockValue) AsInt16(name string) int16     { return v.Called(name).Get(0).(int16) }
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

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
type mockValueBuilder struct {
	istructs.IValueBuilder
	mock.Mock
}

func (b *mockValueBuilder) PutInt8(name string, value int8)                  { b.Called(name, value) }
func (b *mockValueBuilder) PutInt16(name string, value int16)                { b.Called(name, value) }
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
func (b *mockValueBuilder) BuildValue() istructs.IStateValue                 { return nil }
func (b *mockValueBuilder) Equal(src istructs.IStateValueBuilder) bool       { return false }
