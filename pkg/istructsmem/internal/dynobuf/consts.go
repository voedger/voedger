/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
)

var dataKindToDynoFieldType = map[appdef.DataKind]dynobuffers.FieldType{
	appdef.DataKind_null:     dynobuffers.FieldTypeUnspecified,
	appdef.DataKind_int8:     dynobuffers.FieldTypeByte,  // #3434 [small integers : int8]
	appdef.DataKind_int16:    dynobuffers.FieldTypeInt16, // #3434 [small integers : int16]
	appdef.DataKind_int32:    dynobuffers.FieldTypeInt32,
	appdef.DataKind_int64:    dynobuffers.FieldTypeInt64,
	appdef.DataKind_float32:  dynobuffers.FieldTypeFloat32,
	appdef.DataKind_float64:  dynobuffers.FieldTypeFloat64,
	appdef.DataKind_bytes:    dynobuffers.FieldTypeByte,
	appdef.DataKind_string:   dynobuffers.FieldTypeString,
	appdef.DataKind_QName:    dynobuffers.FieldTypeByte, // two fixed bytes LittleEndian
	appdef.DataKind_bool:     dynobuffers.FieldTypeBool,
	appdef.DataKind_RecordID: dynobuffers.FieldTypeInt64,
	appdef.DataKind_Record:   dynobuffers.FieldTypeByte,
	appdef.DataKind_Event:    dynobuffers.FieldTypeByte,
}

const (
	viewPartKeySuffix   = "$pkey"
	viewClustColsSuffix = "$ccols"
)
