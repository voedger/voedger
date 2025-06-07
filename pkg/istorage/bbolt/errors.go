/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Alisher Nurmanov
 */

package bbolt

import "errors"

var (
	ErrDataBucketNotFound = errors.New("data bucket not found")
	ErrTTLBucketNotFound  = errors.New("ttl bucket not found")
)
