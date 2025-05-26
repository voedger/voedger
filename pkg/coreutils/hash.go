/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import "hash/fnv"

// returns FNV-1 hash as int64
func HashBytes(b []byte) int64 {
	fnvHash := fnv.New64()
	if _, err := fnvHash.Write(b); err != nil {
		// notest
		panic(err)
	}
	return int64(fnvHash.Sum64()) // nolint G115 (possible data loss is not a problem since is used as bucket key)
}

func LoginHash(login string) int64 {
	return HashBytes([]byte(login))
}
