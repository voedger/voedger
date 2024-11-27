/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

import (
	"encoding/binary"
)

// nolint: revive
func (t *PersistentBLOBKeyType) Bytes() []byte {
	res := make([]byte, 28)
	binary.LittleEndian.PutUint64(res, blobPKPrefix)
	binary.LittleEndian.PutUint32(res[8:], t.AppID)
	binary.LittleEndian.PutUint64(res[12:], uint64(t.WSID))
	binary.LittleEndian.PutUint64(res[20:], uint64(t.ID))
	return res
}

// nolint: revive
func (t *TempBLOBKeyType) Bytes() []byte {
	res := make([]byte, 20)
	binary.LittleEndian.PutUint64(res, blobPKPrefix)
	binary.LittleEndian.PutUint32(res[8:], t.AppID)
	binary.LittleEndian.PutUint64(res[12:], uint64(t.WSID))
	res = append(res, []byte(t.SUUID)...)
	return res
}
