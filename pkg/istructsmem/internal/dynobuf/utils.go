/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package dynobuf

import (
	"github.com/untillpro/dynobuffers"
	"github.com/untillpro/voedger/pkg/schemas"
)

// Converts schemas DataKind to dynobuffers FieldType
func DataKindToFieldType(kind schemas.DataKind) dynobuffers.FieldType {
	return dataKindToDynoFieldType[kind]
}

// Converts dynobuffers FieldType to string
func FieldTypeToString(ft dynobuffers.FieldType) string {
	return dynobufferFieldTypeToStr[ft]
}
