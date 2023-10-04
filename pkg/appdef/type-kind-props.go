/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Type kind properties
var typeKindProps = map[TypeKind]struct {
	fieldKinds     map[DataKind]bool
	systemFields   map[string]bool
	containerKinds map[TypeKind]bool
}{
	TypeKind_null: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
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
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: true,
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
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: true,
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
		systemFields: map[string]bool{
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
		systemFields: map[string]bool{
			SystemField_ID:       true,
			SystemField_QName:    true,
			SystemField_IsActive: true,
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
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  true,
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
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  true,
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
		systemFields: map[string]bool{
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
		systemFields: map[string]bool{
			SystemField_ID:        true,
			SystemField_QName:     true,
			SystemField_ParentID:  true,
			SystemField_Container: true,
			SystemField_IsActive:  true,
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
		systemFields: map[string]bool{
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
		systemFields: map[string]bool{
			SystemField_QName: true,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_Element: true,
		},
	},
	TypeKind_Element: {
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
		systemFields: map[string]bool{
			SystemField_QName:     true,
			SystemField_Container: true,
		},
		containerKinds: map[TypeKind]bool{
			TypeKind_Element: true,
		},
	},
	TypeKind_Query: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_Command: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
	TypeKind_Workspace: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[TypeKind]bool{},
	},
}
