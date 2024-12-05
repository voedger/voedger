/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistentBLOBKeyType_Bytes(t *testing.T) {
	key := &PersistentBLOBKeyType{
		ClusterAppID: 1234,
		WSID:         5678,
		BlobID:       91011,
	}
	expected := make([]byte, 28)
	binary.LittleEndian.PutUint64(expected, uint64(blobPrefix_persistent))
	binary.LittleEndian.PutUint32(expected[8:], key.ClusterAppID)
	binary.LittleEndian.PutUint64(expected[12:], uint64(key.WSID))
	binary.LittleEndian.PutUint64(expected[20:], uint64(key.BlobID))

	require.Equal(t, expected, key.Bytes())
	require.Equal(t, "91011", key.ID())
}

func TestTempBLOBKeyType_Bytes(t *testing.T) {
	key := &TempBLOBKeyType{
		ClusterAppID: 1234,
		WSID:         5678,
		SUUID:        "test-suuid",
	}
	expected := make([]byte, 20, 20+len(key.SUUID))
	binary.LittleEndian.PutUint64(expected, uint64(blobPrefix_temporary))
	binary.LittleEndian.PutUint32(expected[8:], key.ClusterAppID)
	binary.LittleEndian.PutUint64(expected[12:], uint64(key.WSID))
	expected = append(expected, []byte(key.SUUID)...)

	require.Equal(t, expected, key.Bytes())
	require.Equal(t, "test-suuid", key.ID())
}
