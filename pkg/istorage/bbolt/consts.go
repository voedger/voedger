/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author: Alisher Nurmanov
 */

package bbolt

import "time"

const (
	ttlBucketName   = "ttlBucket"
	dataBucketName  = "dataBucket"
	cleanupInterval = time.Hour
)

// bolt cannot use empty keys so we declare nullKey
var nullKey = []byte{0}
