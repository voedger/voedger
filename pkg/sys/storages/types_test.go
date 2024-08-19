/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var (
	testRecordQName1      = appdef.NewQName("test", "record1")
	testRecordQName2      = appdef.NewQName("test", "record2")
	testRecordQName3      = appdef.NewQName("test", "record3")
	testViewRecordQName1  = appdef.NewQName("test", "viewRecord1")
	testViewRecordQName2  = appdef.NewQName("test", "viewRecord2")
	testWSQName           = appdef.NewQName("test", "testWS")
	testWSDescriptorQName = appdef.NewQName("test", "testWSDescriptor")
)

type nilViewRecords struct {
	istructs.IViewRecords
}

func put(fieldName string, kind appdef.DataKind, rr istructs.IRowReader, rw istructs.IRowWriter) {
	switch kind {
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
