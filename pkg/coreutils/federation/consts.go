/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import "github.com/voedger/voedger/pkg/iblobstorage"

var (
	TemporaryBLOB_URLTTLToDurationLs = map[string]iblobstorage.DurationType{
		"1d": iblobstorage.DurationType_1Day,
	}
	TemporaryBLOBDurationToURLTTL = map[iblobstorage.DurationType]string{
		iblobstorage.DurationType_1Day: "1d",
	}
)
