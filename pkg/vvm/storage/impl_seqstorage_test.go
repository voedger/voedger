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
	baseSeqID := isequencer.SeqID(istructs.QNameIDRecordIDSequence)

	tests := []struct {
		name  string
		appID isequencer.ClusterAppID
		wsid  isequencer.WSID
		seqID isequencer.SeqID
		num   isequencer.Number
	}{
		{
			name:  "basic operations",
			appID: baseAppID,
			wsid:  baseWsid,
			seqID: baseSeqID,
			num:   42,
		},
		{
			name:  "different appID",
			appID: istructs.ClusterApps[istructs.AppQName_test1_app2],
			wsid:  baseWsid,
			seqID: baseSeqID,
			num:   43,
		},
		{
			name:  "different wsid",
			appID: baseAppID,
			wsid:  isequencer.WSID(2),
			seqID: baseSeqID,
			num:   44,
		},
		{
			name:  "different seqID",
			appID: baseAppID,
			wsid:  baseWsid,
			seqID: isequencer.SeqID(istructs.QNameIDRecordIDSequence + 1),
			num:   45,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify value doesn't exist before Put
			ok, num, err := seqStorage.Get(tt.appID+isequencer.ClusterAppID(i), tt.wsid+isequencer.WSID(i), tt.seqID+isequencer.SeqID(i))
			require.NoError(err)
			require.False(ok)
			require.Zero(num)

			// Put value
			batch := []isequencer.SeqValue{{
				Key: isequencer.NumberKey{
					WSID:  tt.wsid + isequencer.WSID(i),
					SeqID: tt.seqID + isequencer.SeqID(i),
				},
				Value: tt.num,
			}}
			err = seqStorage.PutBatch(tt.appID+isequencer.ClusterAppID(i), batch)
			require.NoError(err)

			// Verify value after Put
			ok, num, err = seqStorage.Get(tt.appID+isequencer.ClusterAppID(i), tt.wsid+isequencer.WSID(i), tt.seqID+isequencer.SeqID(i))
			require.NoError(err)
			require.True(ok)
			require.Equal(tt.num, num)

			// check the key structure using the underlying storage
			pKey := []byte{}
			pKey = binary.BigEndian.AppendUint32(pKey, pKeyPrefix_SeqStorage)
			pKey = binary.BigEndian.AppendUint32(pKey, tt.appID+isequencer.ClusterAppID(i))
			cCols := []byte{}
			cCols = binary.BigEndian.AppendUint64(cCols, uint64(tt.wsid+isequencer.WSID(i)))
			cCols = binary.BigEndian.AppendUint16(cCols, uint16(tt.seqID+isequencer.SeqID(i)))
			data := []byte{}
			ok, err = sysVvmAppStorage.Get(pKey, cCols, &data)
			require.NoError(err)
			require.True(ok)
			expectedBytes := []byte{}
			expectedBytes = binary.BigEndian.AppendUint64(expectedBytes, uint64(tt.num))
			require.Equal(expectedBytes, data)
		})
	}

	// Test overwrite separately since it requires two operations
	t.Run("overwrite value", func(t *testing.T) {
		baseAppID := istructs.ClusterApps[istructs.AppQName_test1_app2]
		baseWsid := isequencer.WSID(10)
		baseSeqID := isequencer.SeqID(istructs.QNameIDRecordIDSequence)
		num1 := isequencer.Number(42)
		num2 := isequencer.Number(43)

		// Verify value doesn't exist before Put
		ok, num, err := seqStorage.Get(baseAppID, baseWsid, baseSeqID)
		require.NoError(err)
		require.False(ok)
		require.Zero(num)

		// Put initial value
		batch := []isequencer.SeqValue{{Key: isequencer.NumberKey{WSID: baseWsid, SeqID: baseSeqID}, Value: num1}}
		err = seqStorage.PutBatch(baseAppID, batch)
		require.NoError(err)

		// Overwrite with new value
		batch = []isequencer.SeqValue{{Key: isequencer.NumberKey{WSID: baseWsid, SeqID: baseSeqID}, Value: num2}}
		err = seqStorage.PutBatch(baseAppID, batch)
		require.NoError(err)

		// Verify new value
		ok, actualNum, err := seqStorage.Get(baseAppID, baseWsid, baseSeqID)
		require.NoError(err)
		require.True(ok)
		require.Equal(num2, actualNum)
	})

	t.Run("panic on try to write PLogOffsetSequence", func(t *testing.T) {
		batch := []isequencer.SeqValue{{
			Key: isequencer.NumberKey{
				WSID:  1,
				SeqID: isequencer.SeqID(istructs.QNameIDPLogOffsetSequence),
			},
			Value: 42,
		}}
		require.Panics(func() { seqStorage.PutBatch(istructs.ClusterAppID(1), batch) })
	})
}

func TestPutPLogOffset(t *testing.T) {
	require := require.New(t)
	appStorageProvider := provider.Provide(mem.Provide(coreutils.MockTime))
	sysVvmAppStorage, err := appStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(err)
	seqStorage := NewVVMSeqStorageAdapter(sysVvmAppStorage)

	// Base test data
	baseAppID := istructs.ClusterApps[istructs.AppQName_test1_app1]
	pLogOffset := isequencer.PLogOffset(42)

	// initially missing
	ok, num, err := seqStorage.Get(baseAppID, isequencer.WSID(istructs.NullWSID), isequencer.SeqID(istructs.QNameIDPLogOffsetSequence))
	require.NoError(err)
	require.False(ok)
	require.Zero(num)

	// write
	require.NoError(seqStorage.PutPLogOffset(baseAppID, pLogOffset))

	// read
	ok, num, err = seqStorage.Get(baseAppID, isequencer.WSID(istructs.NullWSID), isequencer.SeqID(istructs.QNameIDPLogOffsetSequence))
	require.NoError(err)
	require.True(ok)
	require.Equal(pLogOffset, isequencer.PLogOffset(num))
}
