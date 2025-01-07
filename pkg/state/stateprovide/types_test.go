/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
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

func (b *mapKeyBuilder) String() string {
	bb := new(bytes.Buffer)
	fmt.Fprintf(bb, "storage:%s", b.storage)
	if b.entity != appdef.NullQName {
		fmt.Fprintf(bb, ", entity:%s", b.entity)
	}
	for key, value := range b.data {
		fmt.Fprintf(bb, ", %s:%v", key, value)
	}
	return bb.String()
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
func (b *mapKeyBuilder) PutNumber(string, json.Number)                    { panic(ErrNotSupported) }
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

func TestKeyBuilder(t *testing.T) {
	require := require.New(t)

	k := newMapKeyBuilder(testStorage, appdef.NullQName)

	require.Equal(testStorage, k.storage)
	require.PanicsWithValue(ErrNotSupported, func() { k.PartitionKey() })
	require.PanicsWithValue(ErrNotSupported, func() { k.ClusteringColumns() })
}

func mockedStructs(t *testing.T) (*mockAppStructs, *mockViewRecords) {
	adb := builder.New()

	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(testWSQName)
	wsDesc := wsb.AddCDoc(testWSDescriptorQName)
	wsDesc.AddField(authnz.Field_WSKind, appdef.DataKind_bytes, false)
	wsb.SetDescriptor(testWSDescriptorQName)

	view := wsb.AddView(testViewRecordQName1)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	view = wsb.AddView(testViewRecordQName2)
	view.Key().PartKey().AddField("pkk", appdef.DataKind_int64)
	view.Key().ClustCols().AddField("cck", appdef.DataKind_string)
	view.Value().AddField("vk", appdef.DataKind_string, false)

	app, err := adb.Build()
	require.NoError(t, err)

	mockWorkspaceRecord := &mockRecord{}
	mockWorkspaceRecord.On("AsQName", "WSKind").Return(testWSDescriptorQName)
	mockWorkspaceRecord.On("QName").Return(qNameCDocWorkspaceDescriptor)
	mockedRecords := &mockRecords{}
	mockedRecords.On("GetSingleton", istructs.WSID(1), mock.Anything).Return(mockWorkspaceRecord, nil)

	mockedViews := &mockViewRecords{}
	mockedViews.On("KeyBuilder", testViewRecordQName1).Return(newMapKeyBuilder(sys.Storage_View, testViewRecordQName1))

	appStructs := &mockAppStructs{}
	appStructs.
		On("AppDef").Return(app).
		On("AppQName").Return(testAppQName).
		On("Records").Return(mockedRecords).
		On("Events").Return(&nilEvents{}).
		On("ViewRecords").Return(mockedViews)

	return appStructs, mockedViews
}

func TestBundle(t *testing.T) {
	newKey := func(qname appdef.QName, id istructs.RecordID) (k istructs.IStateKeyBuilder) {
		k = newMapKeyBuilder(sys.Storage_View, qname)
		k.PutRecordID(sys.Storage_Record_Field_ID, id)
		return
	}
	t.Run("put", func(t *testing.T) {
		b := newBundle()

		b.put(newKey(testRecordQName1, istructs.RecordID(1)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(1)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(2)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(1)), state.ApplyBatchItem{})

		require.Equal(t, 3, b.size())
	})
	t.Run("get", func(t *testing.T) {
		b := newBundle()
		b.put(newKey(testRecordQName1, istructs.RecordID(1)), state.ApplyBatchItem{})

		tests := []struct {
			name string
			key  istructs.IStateKeyBuilder
			want bool
		}{
			{
				name: "Should be ok",
				key:  newKey(testRecordQName1, istructs.RecordID(1)),
				want: true,
			},
			{
				name: "Should be not ok",
				key:  newKey(testRecordQName1, istructs.RecordID(2)),
				want: false,
			},
		}
		for _, test := range tests {
			_, ok := b.get(test.key)

			require.Equal(t, test.want, ok)
		}
	})
	t.Run("containsKeysForSameView", func(t *testing.T) {
		require := require.New(t)
		b := newBundle()

		b.put(newKey(testRecordQName1, istructs.RecordID(1)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(2)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(3)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(4)), state.ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(5)), state.ApplyBatchItem{})

		require.Equal(5, b.size(), "initial bundle size")
		require.False(b.containsKeysForSameEntity(newMapKeyBuilder(sys.Storage_View, testRecordQName3)))

		k := newMapKeyBuilder(sys.Storage_View, testRecordQName2)
		require.True(b.containsKeysForSameEntity(k))
		require.Equal(5, b.size(), "remain bundle size")
	})
}
