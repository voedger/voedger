/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"strings"
)

// Is specified type kind may be used in child containers.
func (k TypeKind) ContainerKindAvailable(s TypeKind) bool {
	return structTypeProps(k).containerKinds[s]
}

// Is field with data kind allowed.
func (k TypeKind) FieldKindAvailable(d DataKind) bool {
	return structTypeProps(k).fieldKinds[d]
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
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
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
	fieldKinds     map[DataKind]bool
	systemFields   map[FieldName]bool
	containerKinds map[TypeKind]bool
}

func structTypeProps(k TypeKind) (props structuralTypeProps) {

	var (
		nullProps = structuralTypeProps{
			fieldKinds:     map[DataKind]bool{},
			systemFields:   map[FieldName]bool{},
			containerKinds: map[TypeKind]bool{},
		}

		structs = map[TypeKind]structuralTypeProps{
			TypeKind_GDoc: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:       true,
					SystemField_QName:    true,
					SystemField_IsActive: false, // exists, but not required
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_GRecord: true,
				},
			},
			TypeKind_CDoc: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:       true,
					SystemField_QName:    true,
					SystemField_IsActive: false,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_CRecord: true,
				},
			},
			TypeKind_ODoc: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:    true,
					SystemField_QName: true,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
					TypeKind_ORecord: true,
				},
			},
			TypeKind_WDoc: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:       true,
					SystemField_QName:    true,
					SystemField_IsActive: false,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_WRecord: true,
				},
			},
			TypeKind_GRecord: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:        true,
					SystemField_QName:     true,
					SystemField_ParentID:  true,
					SystemField_Container: true,
					SystemField_IsActive:  false,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_GRecord: true,
				},
			},
			TypeKind_CRecord: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:        true,
					SystemField_QName:     true,
					SystemField_ParentID:  true,
					SystemField_Container: true,
					SystemField_IsActive:  false,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_CRecord: true,
				},
			},
			TypeKind_ORecord: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:        true,
					SystemField_QName:     true,
					SystemField_ParentID:  true,
					SystemField_Container: true,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_ORecord: true,
				},
			},
			TypeKind_WRecord: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_ID:        true,
					SystemField_QName:     true,
					SystemField_ParentID:  true,
					SystemField_Container: true,
					SystemField_IsActive:  false,
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_WRecord: true,
				},
			},
			TypeKind_ViewRecord: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
					DataKind_Record:   true,
					DataKind_Event:    true,
				},
				systemFields: map[FieldName]bool{
					SystemField_QName: true,
				},
				containerKinds: map[TypeKind]bool{},
			},
			TypeKind_Object: {
				fieldKinds: map[DataKind]bool{
					DataKind_int32:    true,
					DataKind_int64:    true,
					DataKind_float32:  true,
					DataKind_float64:  true,
					DataKind_bytes:    true,
					DataKind_string:   true,
					DataKind_QName:    true,
					DataKind_bool:     true,
					DataKind_RecordID: true,
				},
				systemFields: map[FieldName]bool{
					SystemField_QName:     true,
					SystemField_Container: false, // exists, but required for nested (child) objects only
				},
				containerKinds: map[TypeKind]bool{
					TypeKind_Object: true,
				},
			},
		}
	)

	props = nullProps
	if p, ok := structs[k]; ok {
		props = p
	}
	return props
}
