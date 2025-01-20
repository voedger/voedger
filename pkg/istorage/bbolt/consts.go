/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Alisher Nurmanov
 */

package bbolt

import "time"

const (
	ttlBucketName   = "ttlBucket"
	dataBucketName  = "dataBucket"
	cleanupInterval = time.Hour
)
