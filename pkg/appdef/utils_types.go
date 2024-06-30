/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
	"github.com/voedger/voedger/pkg/utils/utils"
)

// Is specified type kind may be used in child containers.
func (k TypeKind) ContainerKindAvailable(s TypeKind) bool {
	return structTypeProps(k).containerKinds.Contains(s)
}

// Is field with data kind allowed.
func (k TypeKind) FieldKindAvailable(d DataKind) bool {
	return structTypeProps(k).fieldKinds.Contains(d)
}

// Is specified system field exists and required.
func (k TypeKind) HasSystemField(f FieldName) (exists, required bool) {
	required, exists = structTypeProps(k).systemFields[f]
	return exists, required
}

func (k TypeKind) MarshalText() ([]byte, error) {
	var s string
	if k < TypeKind_count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an TypeKind in human-readable form, without `TypeKind_` prefix,
// suitable for debugging or error messages
func (k TypeKind) TrimString() string {
	const pref = "TypeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Structural type kind properties
type structuralTypeProps struct {
	fieldKinds     set.Set[DataKind]
	systemFields   map[FieldName]bool
	containerKinds set.Set[TypeKind]
}

var (
	nullStructProps = &structuralTypeProps{
		fieldKinds:     set.Empty[DataKind](),
		systemFields:   map[FieldName]bool{},
		containerKinds: set.Empty[TypeKind](),
	}

	structFieldKinds = set.From(
		DataKind_int32,
		DataKind_int64,
		DataKind_float32,
		DataKind_float64,
		DataKind_bytes,
		DataKind_string,
		DataKind_QName,
		DataKind_bool,
		DataKind_RecordID,
	)

	typeKindStructProps = map[TypeKind]*structuralTypeProps{
		TypeKind_GDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false, // exists, but not required
			},
			containerKinds: set.From(
				TypeKind_GRecord,
			),
		},
		TypeKind_CDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false,
			},
			containerKinds: set.From(
				TypeKind_CRecord,
			),
		},
		TypeKind_ODoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:    true,
				SystemField_QName: true,
			},
			containerKinds: set.From(
				TypeKind_ODoc, // #19322!: ODocs should be able to contain ODocs
				TypeKind_ORecord,
			),
		},
		TypeKind_WDoc: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:       true,
				SystemField_QName:    true,
				SystemField_IsActive: false,
			},
			containerKinds: set.From(
				TypeKind_WRecord,
			),
		},
		TypeKind_GRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_GRecord,
			),
		},
		TypeKind_CRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_CRecord,
			),
		},
		TypeKind_ORecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
			},
			containerKinds: set.From(
				TypeKind_ORecord,
			),
		},
		TypeKind_WRecord: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_ID:        true,
				SystemField_QName:     true,
				SystemField_ParentID:  true,
				SystemField_Container: true,
				SystemField_IsActive:  false,
			},
			containerKinds: set.From(
				TypeKind_WRecord,
			),
		},
		TypeKind_ViewRecord: {
			fieldKinds: set.From(
				DataKind_int32,
				DataKind_int64,
				DataKind_float32,
				DataKind_float64,
				DataKind_bytes,
				DataKind_string,
				DataKind_QName,
				DataKind_bool,
				DataKind_RecordID,
				DataKind_Record,
				DataKind_Event,
			),
			systemFields: map[FieldName]bool{
				SystemField_QName: true,
			},
			containerKinds: set.Empty[TypeKind](),
		},
		TypeKind_Object: {
			fieldKinds: structFieldKinds,
			systemFields: map[FieldName]bool{
				SystemField_QName:     true,
				SystemField_Container: false, // exists, but required for nested (child) objects only
			},
			containerKinds: set.From(
				TypeKind_Object,
			),
		},
	}
)

func structTypeProps(k TypeKind) *structuralTypeProps {
	props := nullStructProps
	if p, ok := typeKindStructProps[k]; ok {
		props = p
	}
	return props
}
