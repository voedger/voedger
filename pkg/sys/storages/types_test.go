/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package storages

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func TestBundle(t *testing.T) {
	newKey := func(qname appdef.QName, id istructs.RecordID) (k istructs.IStateKeyBuilder) {
		k = &viewKeyBuilder{
			IKeyBuilder: &mapKeyBuilder{data: make(map[string]interface{})},
			view:        qname,
		}
		k.PutRecordID(sys.Storage_Record_Field_ID, id)
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

		require.False(b.containsKeysForSameEntity(&viewKeyBuilder{view: testRecordQName3}))

		k := &viewKeyBuilder{view: testRecordQName2}

		require.True(b.containsKeysForSameEntity(k))
		require.Equal(5, b.size(), "remain bundle size")
	})
}
