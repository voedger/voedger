/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package istoragecache

import "time"

const stackKeySize = 512

var (
	// maxTTL is the maximum TTL value that can be set for a key in cache.
	maxTTL = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC).UnixMilli()
)
