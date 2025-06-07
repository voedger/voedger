/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/binary"
	"time"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

type DataWithExpiration struct {
	ExpireAt int64
	Data     []byte
}

// first 8 bytes - ExpireAt, then - data
func (d DataWithExpiration) ToBytes() []byte {
	res := make([]byte, 0, len(d.Data)+utils.Uint64Size)
	res = binary.BigEndian.AppendUint64(res, uint64(d.ExpireAt)) // nolint G115
	res = append(res, d.Data...)

	return res
}

func ReadWithExpiration(data []byte) DataWithExpiration {
	return DataWithExpiration{
		ExpireAt: int64(binary.BigEndian.Uint64(data[:utils.Uint64Size])), // nolint G115
		Data:     data[utils.Uint64Size:],
	}
}

func (d DataWithExpiration) IsExpired(now time.Time) bool {
	return d.ExpireAt > 0 && !now.Before(time.UnixMilli(d.ExpireAt))
}

func (d DataWithExpiration) Update(data []byte) DataWithExpiration {
	d.ExpireAt = int64(binary.BigEndian.Uint64(data[:utils.Uint64Size])) // nolint G115
	d.Data = data[utils.Uint64Size:]
	return d
}
