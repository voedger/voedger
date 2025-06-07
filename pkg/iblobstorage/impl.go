/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// nolint: revive
func (t *PersistentBLOBKeyType) Bytes() []byte {
	res := make([]byte, 28)
	binary.LittleEndian.PutUint64(res, uint64(blobPrefix_persistent))
	binary.LittleEndian.PutUint32(res[8:], t.ClusterAppID)
	binary.LittleEndian.PutUint64(res[12:], uint64(t.WSID))
	binary.LittleEndian.PutUint64(res[20:], uint64(t.BlobID))
	return res
}

func (t *PersistentBLOBKeyType) ID() string {
	return utils.UintToString(t.BlobID)
}


func (t *PersistentBLOBKeyType) IsPersistent() bool {
	return true
}

// nolint: revive
func (t *TempBLOBKeyType) Bytes() []byte {
	res := make([]byte, 20, 20+len(t.SUUID))
	binary.LittleEndian.PutUint64(res, uint64(blobPrefix_temporary))
	binary.LittleEndian.PutUint32(res[8:], t.ClusterAppID)
	binary.LittleEndian.PutUint64(res[12:], uint64(t.WSID))
	res = append(res, []byte(t.SUUID)...)
	return res
}

func (t *TempBLOBKeyType) ID() string {
	return string(t.SUUID)
}

func (t *TempBLOBKeyType) IsPersistent() bool {
	return false
}
