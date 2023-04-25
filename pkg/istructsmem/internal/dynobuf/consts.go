/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/schemas"
)

var dataKindToDynoFieldType = map[schemas.DataKind]dynobuffers.FieldType{
	schemas.DataKind_null:     dynobuffers.FieldTypeUnspecified,
	schemas.DataKind_int32:    dynobuffers.FieldTypeInt32,
	schemas.DataKind_int64:    dynobuffers.FieldTypeInt64,
	schemas.DataKind_float32:  dynobuffers.FieldTypeFloat32,
	schemas.DataKind_float64:  dynobuffers.FieldTypeFloat64,
	schemas.DataKind_bytes:    dynobuffers.FieldTypeByte,
	schemas.DataKind_string:   dynobuffers.FieldTypeString,
	schemas.DataKind_QName:    dynobuffers.FieldTypeByte, // two fixed bytes LittleEndian
	schemas.DataKind_bool:     dynobuffers.FieldTypeBool,
	schemas.DataKind_RecordID: dynobuffers.FieldTypeInt64,
	schemas.DataKind_Record:   dynobuffers.FieldTypeByte,
	schemas.DataKind_Event:    dynobuffers.FieldTypeByte,
}

var dynobufferFieldTypeToStr = map[dynobuffers.FieldType]string{
	dynobuffers.FieldTypeUnspecified: "null",
	dynobuffers.FieldTypeInt32:       "int32",
	dynobuffers.FieldTypeInt64:       "int64",
	dynobuffers.FieldTypeFloat32:     "float32",
	dynobuffers.FieldTypeFloat64:     "float64",
	dynobuffers.FieldTypeString:      "string",
	dynobuffers.FieldTypeBool:        "bool",
	dynobuffers.FieldTypeByte:        "[]byte",
}
