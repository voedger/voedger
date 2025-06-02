/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/set"
	"github.com/voedger/voedger/pkg/istructs"
)

// Application config
const (
	// events per application plog cache, see [#455](https://github.com/voedger/voedger/issues/455#:~:text=Currently%2C%2010000%20must%20be%20used)
	DefaultPLogEventCacheSize = 10 * 1000
)

/* internal package constants */

// nullResource is result then resource not found
var nullResource istructs.IResource = newNullResource()

const (
	// byte codec versions
	codec_RawDynoBuffer = byte(0x00) + iota
	codec_RDB_1         // + row system fields mask
	codec_RDB_2         // + CUD row emptied fields

	// !do not forget to actualize last codec version!
	codec_LastVersion = codec_RDB_2
)

// maskString is character to mask values in string cell, used for obfuscate unlogged command arguments data
const maskString = "*"

// constants to split IDs to two-parts key — partition key and clustering columns
const (
	partitionBits        = 12
	lowMask              = uint16((1 << partitionBits) - 1)
	partitionRecordCount = 1 << partitionBits
)

// maxGetBatchRecordCount is maximum records that can be retrieved by ReadBatch GetBatch
const maxGetBatchRecordCount = 256

// system fields mask values
const (
	sfm_ID        = uint16(1 << 0)
	sfm_ParentID  = uint16(1 << 1)
	sfm_Container = uint16(1 << 2)
	sfm_IsActive  = uint16(1 << 3)
)

// rate limits function name formats, see GetFunctionRateLimitName
var funcRateLimitNameFmt = [istructs.RateLimitKind_FakeLast]string{
	"func_%s_byApp",
	"func_%s_byWS",
	"func_%s_byID",
}

// Set of type kinds that stored directly in wlog
var recordsInWLog = set.FromRO(appdef.TypeKind_ODoc, appdef.TypeKind_ORecord)
