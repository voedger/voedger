/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/schemas"
)

var dataKindToDynoFieldType = map[schemas.DataKind]dynobuffers.FieldType{
	istructs.DataKind_null:     dynobuffers.FieldTypeUnspecified,
	istructs.DataKind_int32:    dynobuffers.FieldTypeInt32,
	istructs.DataKind_int64:    dynobuffers.FieldTypeInt64,
	istructs.DataKind_float32:  dynobuffers.FieldTypeFloat32,
	istructs.DataKind_float64:  dynobuffers.FieldTypeFloat64,
	istructs.DataKind_bytes:    dynobuffers.FieldTypeByte,
	istructs.DataKind_string:   dynobuffers.FieldTypeString,
	istructs.DataKind_QName:    dynobuffers.FieldTypeByte, // two fixed bytes LittleEndian
	istructs.DataKind_bool:     dynobuffers.FieldTypeBool,
	istructs.DataKind_RecordID: dynobuffers.FieldTypeInt64,
	istructs.DataKind_Record:   dynobuffers.FieldTypeByte,
	istructs.DataKind_Event:    dynobuffers.FieldTypeByte,
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
