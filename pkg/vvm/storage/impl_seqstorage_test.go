/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package storage

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/isequencer"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
)

func TestSeqStorage(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqStorage := NewVVMSeqStorageAdapter(sysVvmAppStorage)

	// Base test data
	baseAppID := istructs.ClusterApps[istructs.AppQName_test1_app1]
	baseWsid := isequencer.WSID(1)
	baseSeqID := isequencer.SeqID(istructs.QNameIDCRecordIDSequence)

	tests := []struct {
		name     string
		appID    istructs.ClusterAppID
		wsid     isequencer.WSID
		seqID    isequencer.SeqID
		value    []byte
		expected []byte
	}{
		{
			name:     "basic operations",
			appID:    baseAppID,
			wsid:     baseWsid,
			seqID:    baseSeqID,
			value:    []byte{1, 2, 3},
			expected: []byte{1, 2, 3},
		},
		{
			name:     "empty value",
			appID:    baseAppID,
			wsid:     baseWsid,
			seqID:    baseSeqID,
			value:    []byte{},
			expected: []byte{},
		},
		{
			name:     "nil value",
			appID:    baseAppID,
			wsid:     baseWsid,
			seqID:    baseSeqID,
			value:    nil,
			expected: []byte{},
		},
		{
			name:  "large value",
			appID: baseAppID,
			wsid:  baseWsid,
			seqID: baseSeqID,
			value: func() []byte {
				v := make([]byte, 1024*1024) // 1MB
				for i := range v {
					v[i] = byte(i % 256)
				}
				return v
			}(),
			expected: func() []byte {
				v := make([]byte, 1024*1024) // 1MB
				for i := range v {
					v[i] = byte(i % 256)
				}
				return v
			}(),
		},
		{
			name:     "different appID",
			appID:    istructs.ClusterApps[istructs.AppQName_test1_app2],
			wsid:     baseWsid,
			seqID:    baseSeqID,
			value:    []byte{4, 5, 6},
			expected: []byte{4, 5, 6},
		},
		{
			name:     "different wsid",
			appID:    baseAppID,
			wsid:     isequencer.WSID(2),
			seqID:    baseSeqID,
			value:    []byte{7, 8, 9},
			expected: []byte{7, 8, 9},
		},
		{
			name:     "different seqID",
			appID:    baseAppID,
			wsid:     baseWsid,
			seqID:    isequencer.SeqID(istructs.QNameIDCRecordIDSequence + 1),
			value:    []byte{10, 11, 12},
			expected: []byte{10, 11, 12},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify value doesn't exist before Put
			data := []byte{}
			ok, err := seqStorage.Get(tt.appID+istructs.ClusterAppID(i), tt.wsid+isequencer.WSID(i), tt.seqID+isequencer.SeqID(i), &data)
			require.NoError(err)
			require.False(ok)
			require.Empty(data)

			// Put value
			err = seqStorage.Put(tt.appID+istructs.ClusterAppID(i), tt.wsid+isequencer.WSID(i), tt.seqID+isequencer.SeqID(i), tt.value)
			require.NoError(err)

			// Verify value after Put
			data = []byte{}
			ok, err = seqStorage.Get(tt.appID+istructs.ClusterAppID(i), tt.wsid+isequencer.WSID(i), tt.seqID+isequencer.SeqID(i), &data)
			require.NoError(err)
			require.True(ok)
			require.Equal(tt.expected, data)

			// check the key structure using the underlying storage
			pKey := []byte{}
			pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage)
			pKey = binary.BigEndian.AppendUint32(pKey, tt.appID+istructs.ClusterAppID(i))
			cCols := []byte{}
			cCols = binary.BigEndian.AppendUint64(cCols, uint64(tt.wsid+isequencer.WSID(i)))
			cCols = binary.BigEndian.AppendUint16(cCols, uint16(tt.seqID+isequencer.SeqID(i)))
			data = []byte{}
			ok, err = sysVvmAppStorage.Get(pKey, cCols, &data)
			require.NoError(err)
			require.True(ok)
			require.Equal(tt.expected, data)
		})
	}

	// Test overwrite separately since it requires two operations
	t.Run("overwrite value", func(t *testing.T) {
		baseAppID := istructs.ClusterApps[istructs.AppQName_test1_app2]
		baseWsid := isequencer.WSID(10)
		baseSeqID := isequencer.SeqID(istructs.QNameIDCRecordIDSequence)
		value1 := []byte{1, 2, 3}
		value2 := []byte{4, 5, 6}

		// Verify value doesn't exist before Put
		data := []byte{}
		ok, err := seqStorage.Get(baseAppID, baseWsid, baseSeqID, &data)
		require.NoError(err)
		require.False(ok)
		require.Empty(data)

		// Put initial value
		err = seqStorage.Put(baseAppID, baseWsid, baseSeqID, value1)
		require.NoError(err)

		// Overwrite with new value
		err = seqStorage.Put(baseAppID, baseWsid, baseSeqID, value2)
		require.NoError(err)

		// Verify new value
		data = []byte{}
		ok, err = seqStorage.Get(baseAppID, baseWsid, baseSeqID, &data)
		require.NoError(err)
		require.True(ok)
		require.Equal(value2, data)
	})
}
