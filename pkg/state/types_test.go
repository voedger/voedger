/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package state

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestBundle(t *testing.T) {
	newKey := func(qname appdef.QName, id istructs.RecordID) (k istructs.IStateKeyBuilder) {
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

func TestKeyBuilder(t *testing.T) {
	require := require.New(t)

	k := newKeyBuilder(testStorage, appdef.NullQName)

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
