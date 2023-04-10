/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"testing"

	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/stretchr/testify/require"
)

func TestBundle(t *testing.T) {
	newKey := func(qname istructs.QName, id istructs.RecordID) (k istructs.IStateKeyBuilder) {
		k = &viewRecordsKeyBuilder{
			IKeyBuilder: &keyBuilder{data: make(map[string]interface{})},
			view:        qname,
		}
		k.PutRecordID(Field_ID, id)
		return
	}
	t.Run("put", func(t *testing.T) {
		b := newBundle()

		b.put(newKey(testRecordQName1, istructs.RecordID(1)), ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(1)), ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(2)), ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(1)), ApplyBatchItem{})

		require.Equal(t, 3, b.size())
	})
	t.Run("get", func(t *testing.T) {
		b := newBundle()
		b.put(newKey(testRecordQName1, istructs.RecordID(1)), ApplyBatchItem{})

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

		b.put(newKey(testRecordQName1, istructs.RecordID(1)), ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(2)), ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(3)), ApplyBatchItem{})
		b.put(newKey(testRecordQName2, istructs.RecordID(4)), ApplyBatchItem{})
		b.put(newKey(testRecordQName1, istructs.RecordID(5)), ApplyBatchItem{})

		require.Equal(5, b.size(), "initial bundle size")

		require.False(b.containsKeysForSameEntity(&viewRecordsKeyBuilder{view: testRecordQName3}))

		k := &viewRecordsKeyBuilder{view: testRecordQName2}

		require.True(b.containsKeysForSameEntity(k))
		require.Equal(5, b.size(), "remain bundle size")
	})
}
func TestRowWriter(t *testing.T) {
	require := require.New(t)

	rw := newRowWriter()

	rw.PutInt32("Int32", int32(1))
	rw.PutInt64("Int64", int64(2))
	rw.PutFloat32("Float32", float32(3.1))
	rw.PutFloat64("Float64", 4.2)
	rw.PutBytes("Bytes", []byte{5})
	rw.PutString("String", "string")
	rw.PutQName("QName", testRecordQName1)
	rw.PutBool("Bool", true)
	rw.PutRecordID("RecordID", istructs.RecordID(6))

	require.Equal(int32(1), rw.data["Int32"])
	require.Equal(int64(2), rw.data["Int64"])
	require.Equal(float32(3.1), rw.data["Float32"])
	require.Equal(4.2, rw.data["Float64"])
	require.Equal([]byte{5}, rw.data["Bytes"])
	require.Equal("string", rw.data["String"])
	require.Equal(testRecordQName1, rw.data["QName"])
	require.Equal(true, rw.data["Bool"])
	require.Equal(istructs.RecordID(6), rw.data["RecordID"])
	require.PanicsWithValue(ErrNotSupported, func() { rw.PutNumber("", 0.0) })
	require.PanicsWithValue(ErrNotSupported, func() { rw.PutChars("", "") })
}
func TestKeyBuilder(t *testing.T) {
	require := require.New(t)

	k := newKeyBuilder(testStorage, istructs.NullQName)

	require.Equal(testStorage, k.storage)
	require.PanicsWithValue(ErrNotSupported, func() { k.PartitionKey() })
	require.PanicsWithValue(ErrNotSupported, func() { k.ClusteringColumns() })
}
func TestHttpStorageKeyBuilder_headers(t *testing.T) {
	require := require.New(t)
	k := newHTTPStorageKeyBuilder()
	k.PutString(Field_Header, "key: hello:world")

	headers := k.headers

	require.Equal("hello:world", headers["key"])
}
