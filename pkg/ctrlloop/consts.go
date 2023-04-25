/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import "time"

const (
	DedupInRetryInterval     = 100 * time.Millisecond
	ReportInterval           = 10 * time.Millisecond
	DedupScheduleInterval    = 10 * time.Second
	MaxReportAttemptNumber   = 3
	keySerialNumberLogFormat = "key: %v, serialNumber: %d"
	keyLogFormat             = "key: %v"
)
