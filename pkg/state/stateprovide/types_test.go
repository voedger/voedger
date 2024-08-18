/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

type mapKeyBuilder struct {
	data    map[string]interface{}
	storage appdef.QName
	entity  appdef.QName
}

func newMapKeyBuilder(storage, entity appdef.QName) *mapKeyBuilder {
	return &mapKeyBuilder{
		data:    make(map[string]interface{}),
		storage: storage,
		entity:  entity,
	}
}

func (b *mapKeyBuilder) Storage() appdef.QName                            { return b.storage }
func (b *mapKeyBuilder) Entity() appdef.QName                             { return b.entity }
func (b *mapKeyBuilder) PutInt32(name string, value int32)                { b.data[name] = value }
func (b *mapKeyBuilder) PutInt64(name string, value int64)                { b.data[name] = value }
func (b *mapKeyBuilder) PutFloat32(name string, value float32)            { b.data[name] = value }
func (b *mapKeyBuilder) PutFloat64(name string, value float64)            { b.data[name] = value }
func (b *mapKeyBuilder) PutBytes(name string, value []byte)               { b.data[name] = value }
func (b *mapKeyBuilder) PutString(name string, value string)              { b.data[name] = value }
func (b *mapKeyBuilder) PutQName(name string, value appdef.QName)         { b.data[name] = value }
func (b *mapKeyBuilder) PutBool(name string, value bool)                  { b.data[name] = value }
func (b *mapKeyBuilder) PutRecordID(name string, value istructs.RecordID) { b.data[name] = value }
func (b *mapKeyBuilder) PutNumber(string, float64)                        { panic(ErrNotSupported) }
func (b *mapKeyBuilder) PutChars(string, string)                          { panic(ErrNotSupported) }
func (b *mapKeyBuilder) PutFromJSON(j map[string]any)                     { maps.Copy(b.data, j) }
func (b *mapKeyBuilder) PartitionKey() istructs.IRowWriter                { panic(ErrNotSupported) }
func (b *mapKeyBuilder) ClusteringColumns() istructs.IRowWriter           { panic(ErrNotSupported) }
func (b *mapKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	kb, ok := src.(*mapKeyBuilder)
	if !ok {
		return false
	}
	if b.storage != kb.storage {
		return false
	}
	if b.entity != kb.entity {
		return false
	}
	if !maps.Equal(b.data, kb.data) {
		return false
	}
	return true
}
func (b *mapKeyBuilder) ToBytes(istructs.WSID) ([]byte, []byte, error) { panic(ErrNotSupported) }

func appStructsFunc(app istructs.IAppStructs) state.AppStructsFunc {
	return func() istructs.IAppStructs {
		return app
	}
}

func TestKeyBuilder(t *testing.T) {
	require := require.New(t)

	k := newMapKeyBuilder(testStorage, appdef.NullQName)

	require.Equal(testStorage, k.storage)
	require.PanicsWithValue(ErrNotSupported, func() { k.PartitionKey() })
	require.PanicsWithValue(ErrNotSupported, func() { k.ClusteringColumns() })
}

func mockedStructs2(t *testing.T, addWsDescriptor bool) (*mockAppStructs, *mockViewRecords) {
	appDef := appdef.New()

	appDef.AddPackage("test", "test.com/test")

	view := appDef.AddView(testViewRecordQName1)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	view = appDef.AddView(testViewRecordQName2)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	mockWorkspaceRecord := &mockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)
	mockedRecords := &mockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	mockedViews := &mockViewRecords{}
	mockedViews.On("KeyBuilder", testViewRecordQName1).Return(newMapKeyBuilder(sys.Storage_View, testViewRecordQName1))

	if addWsDescriptor {
		wsDesc := appDef.AddCDoc(testWSDescriptorQName)
		wsDesc.AddField(authnz.Field_WSKind, appdef.DataKind_bytes, false)
	}

	ws := appDef.AddWorkspace(testWSQName)
	ws.AddType(testViewRecordQName1)
	ws.AddType(testViewRecordQName2)
	ws.SetDescriptor(testWSDescriptorQName)

	app, err := appDef.Build()
	require.NoError(t, err)

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(app).
		On("AppQName").Return(testAppQName).
		On("Records").Return(mockedRecords).
		On("Events").Return(&nilEvents{}).
		On("ViewRecords").Return(mockedViews)

	return appStructs, mockedViews
}

func mockedStructs(t *testing.T) (*mockAppStructs, *mockViewRecords) {
	return mockedStructs2(t, true)
}
