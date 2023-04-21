/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
	"github.com/voedger/voedger/pkg/schemas"
)

// nullResource is result then resource not found
var nullResource istructs.IResource = newNullResource()

const (
	// data kind mnemonics
	dk_null     = "null"
	dk_int32    = "int32"
	dk_int64    = "int64"
	dk_float32  = "float32"
	dk_float64  = "float64"
	dk_bytes    = "[]byte"
	dk_string   = "string"
	dk_QName    = "QName"
	dk_bool     = "bool"
	dk_RecordID = "RecordID"
	dk_Record   = "record"
	dk_Event    = "event"
	// abstracts if unknown type
	dk_Number = "numeric"
	dk_Chars  = "character"
)

// dataKindToStr is map to return data kind mnemonics. Useful to format error messages
var dataKindToStr = map[schemas.DataKind]string{
	schemas.DataKind_null:     dk_null,
	schemas.DataKind_int32:    dk_int32,
	schemas.DataKind_int64:    dk_int64,
	schemas.DataKind_float32:  dk_float32,
	schemas.DataKind_float64:  dk_float64,
	schemas.DataKind_bytes:    dk_bytes,
	schemas.DataKind_string:   dk_string,
	schemas.DataKind_QName:    dk_QName,
	schemas.DataKind_bool:     dk_bool,
	schemas.DataKind_RecordID: dk_RecordID,
	schemas.DataKind_Record:   dk_Record,
	schemas.DataKind_Event:    dk_Event,
}

// const ( // verification kind mnemonics
// 	verify_byEmail = "EMail"
// 	verify_ByPhone = "Phone"
// )

// var verificationKindToStr = map[payloads.VerificationKindType]string{
// 	payloads.VerificationKind_EMail: verify_byEmail,
// 	payloads.VerificationKind_Phone: verify_ByPhone,
// }

const (
	// schema kind mnemonics
	sk_null                         = "null"
	sk_GDoc                         = "GDoc"
	sk_CDoc                         = "CDoc"
	sk_ODoc                         = "ODoc"
	sk_WDoc                         = "WDoc"
	sk_GRecord                      = "GRecord"
	sk_CRecord                      = "CRecord"
	sk_ORecord                      = "ORecord"
	sk_WRecord                      = "WRecord"
	sk_ViewRecord                   = "ViewRecord"
	sk_ViewRecord_PartitionKey      = "ViewRecord_PartitionKey"
	sk_ViewRecord_ClusteringColumns = "ViewRecord_ClusteringColumns"
	sk_ViewRecord_Value             = "ViewRecord_Value"
	sk_Object                       = "Object"
	sk_Element                      = "Element"
	sk_QueryFunction                = "QueryFunction"
	sk_CommandFunction              = "CommandFunction"
)

// shemaKindToStr is map to return schema kind mnemonics. Useful to format error messages
var shemaKindToStr = map[schemas.SchemaKind]string{
	schemas.SchemaKind_null:                         sk_null,
	schemas.SchemaKind_GDoc:                         sk_GDoc,
	schemas.SchemaKind_CDoc:                         sk_CDoc,
	schemas.SchemaKind_ODoc:                         sk_ODoc,
	schemas.SchemaKind_WDoc:                         sk_WDoc,
	schemas.SchemaKind_GRecord:                      sk_GRecord,
	schemas.SchemaKind_CRecord:                      sk_CRecord,
	schemas.SchemaKind_ORecord:                      sk_ORecord,
	schemas.SchemaKind_WRecord:                      sk_WRecord,
	schemas.SchemaKind_ViewRecord:                   sk_ViewRecord,
	schemas.SchemaKind_ViewRecord_PartitionKey:      sk_ViewRecord_PartitionKey,
	schemas.SchemaKind_ViewRecord_ClusteringColumns: sk_ViewRecord_ClusteringColumns,
	schemas.SchemaKind_ViewRecord_Value:             sk_ViewRecord_Value,
	schemas.SchemaKind_Object:                       sk_Object,
	schemas.SchemaKind_Element:                      sk_Element,
	schemas.SchemaKind_QueryFunction:                sk_QueryFunction,
	schemas.SchemaKind_CommandFunction:              sk_CommandFunction,
}

const (
	// byte codec versions
	codec_RawDynoBuffer = byte(0x00) + iota
	codec_RDB_1         // + row system fields mask

	// !do not forget to actualize last codec version!
	codec_LastVersion = codec_RDB_1
)

// maskString is charaster to mask values in string cell, used for obfuscate unlogged command arguments data
const maskString = "*"

// constants to split IDs to two-parts key — partition key and clustering columns
const (
	partitionBits        = 12
	lowMask              = uint16((1 << partitionBits) - 1)
	partitionRecordCount = 1 << partitionBits
)

// maxGetBatchRecordCount is maximum records that can be retrieved by ReadBatch GetBatch
const maxGetBatchRecordCount = 256

// versions of system views
const (
	// sys.QName
	verSysQNames01      vers.VersionValue = 1
	verSysQNamesLastest vers.VersionValue = verSysQNames01

	// sys.Singletons
	verSysSingletons01      vers.VersionValue = 1
	verSysSingletonsLastest vers.VersionValue = verSysSingletons01
)

// system fields mask values
const (
	sfm_ID        = uint16(1 << 0)
	sfm_ParentID  = uint16(1 << 1)
	sfm_Container = uint16(1 << 2)
	sfm_IsActive  = uint16(1 << 3)
)

var nullPrepareArgs = istructs.PrepareArgs{}

// rate limits function name formats, see GetFunctionRateLimitName
var funcRateLimitNameFmt = [istructs.RateLimitKind_FakeLast]string{
	"func_%s_byApp",
	"func_%s_byWS",
	"func_%s_byID",
}
