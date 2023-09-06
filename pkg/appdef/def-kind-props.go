/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Definition kind properties
var defKindProps = map[DefKind]struct {
	fieldKinds     map[DataKind]bool
	systemFields   map[string]bool
	containerKinds map[DefKind]bool
}{
	DefKind_null: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_GDoc: {
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
		containerKinds: map[DefKind]bool{
			DefKind_GRecord: true,
		},
	},
	DefKind_CDoc: {
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
		containerKinds: map[DefKind]bool{
			DefKind_CRecord: true,
		},
	},
	DefKind_ODoc: {
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
		containerKinds: map[DefKind]bool{
			DefKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			DefKind_ORecord: true,
		},
	},
	DefKind_WDoc: {
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
		containerKinds: map[DefKind]bool{
			DefKind_WRecord: true,
		},
	},
	DefKind_GRecord: {
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
		containerKinds: map[DefKind]bool{
			DefKind_GRecord: true,
		},
	},
	DefKind_CRecord: {
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
		containerKinds: map[DefKind]bool{
			DefKind_CRecord: true,
		},
	},
	DefKind_ORecord: {
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
		containerKinds: map[DefKind]bool{
			DefKind_ORecord: true,
		},
	},
	DefKind_WRecord: {
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
		containerKinds: map[DefKind]bool{
			DefKind_WRecord: true,
		},
	},
	DefKind_ViewRecord: {
		fieldKinds:   map[DataKind]bool{},
		systemFields: map[string]bool{},
		containerKinds: map[DefKind]bool{
			DefKind_ViewRecord_PartitionKey:      true,
			DefKind_ViewRecord_ClusteringColumns: true,
			DefKind_ViewRecord_Key:               true,
			DefKind_ViewRecord_Value:             true,
		},
	},
	DefKind_ViewRecord_PartitionKey: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_ViewRecord_ClusteringColumns: {
		fieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_bytes:    true, // last field
			DataKind_string:   true, // last field
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_ViewRecord_Key: {
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
		systemFields: map[string]bool{},
		containerKinds: map[DefKind]bool{
			DefKind_ViewRecord_PartitionKey:      true,
			DefKind_ViewRecord_ClusteringColumns: true,
		},
	},
	DefKind_ViewRecord_Value: {
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
			DataKind_Record:   true, // +
			DataKind_Event:    true, // +
		},
		systemFields: map[string]bool{
			SystemField_QName: true,
		},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_Object: {
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
		containerKinds: map[DefKind]bool{
			DefKind_Element: true,
		},
	},
	DefKind_Element: {
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
		containerKinds: map[DefKind]bool{
			DefKind_Element: true,
		},
	},
	DefKind_Query: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_Command: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
	DefKind_Workspace: {
		fieldKinds:     map[DataKind]bool{},
		systemFields:   map[string]bool{},
		containerKinds: map[DefKind]bool{},
	},
}
