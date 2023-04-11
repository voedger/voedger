/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"github.com/untillpro/voedger/pkg/istructs"
	"github.com/untillpro/voedger/pkg/istructsmem/internal/vers"
	"github.com/untillpro/voedger/pkg/schemas"
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
var dataKindToStr = map[istructs.DataKindType]string{
	istructs.DataKind_null:     dk_null,
	istructs.DataKind_int32:    dk_int32,
	istructs.DataKind_int64:    dk_int64,
	istructs.DataKind_float32:  dk_float32,
	istructs.DataKind_float64:  dk_float64,
	istructs.DataKind_bytes:    dk_bytes,
	istructs.DataKind_string:   dk_string,
	istructs.DataKind_QName:    dk_QName,
	istructs.DataKind_bool:     dk_bool,
	istructs.DataKind_RecordID: dk_RecordID,
	istructs.DataKind_Record:   dk_Record,
	istructs.DataKind_Event:    dk_Event,
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
	istructs.SchemaKind_null:                         sk_null,
	istructs.SchemaKind_GDoc:                         sk_GDoc,
	istructs.SchemaKind_CDoc:                         sk_CDoc,
	istructs.SchemaKind_ODoc:                         sk_ODoc,
	istructs.SchemaKind_WDoc:                         sk_WDoc,
	istructs.SchemaKind_GRecord:                      sk_GRecord,
	istructs.SchemaKind_CRecord:                      sk_CRecord,
	istructs.SchemaKind_ORecord:                      sk_ORecord,
	istructs.SchemaKind_WRecord:                      sk_WRecord,
	istructs.SchemaKind_ViewRecord:                   sk_ViewRecord,
	istructs.SchemaKind_ViewRecord_PartitionKey:      sk_ViewRecord_PartitionKey,
	istructs.SchemaKind_ViewRecord_ClusteringColumns: sk_ViewRecord_ClusteringColumns,
	istructs.SchemaKind_ViewRecord_Value:             sk_ViewRecord_Value,
	istructs.SchemaKind_Object:                       sk_Object,
	istructs.SchemaKind_Element:                      sk_Element,
	istructs.SchemaKind_QueryFunction:                sk_QueryFunction,
	istructs.SchemaKind_CommandFunction:              sk_CommandFunction,
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

// constants for system QNames
const (
	NullQNameID QNameID = 0 + iota
	QNameIDForError
	QNameIDCommandCUD
)

// constants for system container names
const (
	nullContainerNameID containerNameIDType = 0 + iota
	viewPKeyContainerID
	viewCColsContainerID
	viewValueContainerID

	containerNameIDSysLast containerNameIDType = 63
)

// QNames for system views
const (
	QNameIDSysVesions      QNameID = 16 + iota // system view versions
	QNameIDSysQNames                           // application QNames system view
	QNameIDSysContainers                       // application container names view
	QNameIDSysRecords                          // application Records view
	QNameIDSysPLog                             // application PLog view
	QNameIDSysWLog                             // application WLog view
	QNameIDSysSingletonIDs                     // application singletons IDs view

	QNameIDSysLast QNameID = 255
)

// maxGetBatchRecordCount is maximum records that can be retrieved by ReadBatch GetBatch
const maxGetBatchRecordCount = 256

// versions of system views
const (
	// sys.QName
	verSysQNames01      vers.VersionValue = 1
	verSysQNamesLastest vers.VersionValue = verSysQNames01

	// sys.Containers
	verSysContainers01      vers.VersionValue = 1
	verSysContainersLastest vers.VersionValue = verSysContainers01

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
