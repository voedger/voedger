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
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
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

type nilViewRecords struct {
	istructs.IViewRecords
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
	require := require.New(t)
	t.Run("Should be ok", func(t *testing.T) {
		mrw := &coreutils.MockStateValueBuilder{}
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
		mrr := &coreutils.MockKey{}
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
		require.PanicsWithError(fmt.Sprintf("illegal state: field - 'notSupported', kind - '%d': not supported", appdef.DataKind_FakeLast),
			func() { put("notSupported", appdef.DataKind_FakeLast, nil, nil) })
	})
}

// TODO: copy-pasted from pkg/state/stateprovide. Can this be moved to a common package?
func mockedStructs(t *testing.T) (*coreutils.MockAppStructs, *coreutils.MockViewRecords) {
	t.Helper()
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

	app := appDef.MustBuild()

	mockWorkspaceRecord := &coreutils.MockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(appdef.QNameCDocWorkspaceDescriptor)
	mockedRecords := &coreutils.MockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	mockedViews := &coreutils.MockViewRecords{}
	mockedViews.On("KeyBuilder", testViewRecordQName1).Return(&viewKeyBuilder{IKeyBuilder: newUniqKeyBuilder(sys.Storage_View, appdef.NullQName), view: testViewRecordQName1})

	appStructs := &coreutils.MockAppStructs{}
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
