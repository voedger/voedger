/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"regexp"
	"time"

	"github.com/voedger/voedger/pkg/iblobstorage"
)

const federationRetryMaxDuration = 30 * time.Second

var (
	TemporaryBLOB_URLTTLToDurationLs = map[string]iblobstorage.DurationType{
		"1d": iblobstorage.DurationType_1Day,
	}
	TemporaryBLOBDurationToURLTTL = map[iblobstorage.DurationType]string{
		iblobstorage.DurationType_1Day: "1d",
	}
	blobCreatePersistentRespRE = regexp.MustCompile(`"blobID":\s*(\d+)`)
	blobCreateTempRespRE       = regexp.MustCompile(`"blobSUUID":\s*"(.+)"`)
)
