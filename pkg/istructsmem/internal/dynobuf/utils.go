/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/voedger/voedger/pkg/appdef"
)

// Converts appdef.DataKind to dynobuffers.FieldType
func DataKindToFieldType(kind appdef.DataKind) dynobuffers.FieldType {
	return dataKindToDynoFieldType[kind]
}

// Converts dynobuffers FieldType to string
func FieldTypeToString(ft dynobuffers.FieldType) string {
	return dynobufferFieldTypeToStr[ft]
}
