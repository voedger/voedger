/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Definition kind properties
var defKindProps = map[DefKind]struct {
	fieldsAllowed           bool
	availableFieldKinds     map[DataKind]bool
	systemFields            map[string]bool
	containersAllowed       bool
	availableContainerKinds map[DefKind]bool
}{
	DefKind_null: {
		fieldsAllowed:           false,
		availableFieldKinds:     map[DataKind]bool{},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[DefKind]bool{},
	},
	DefKind_GDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_GRecord: true,
		},
	},
	DefKind_CDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_CRecord: true,
		},
	},
	DefKind_ODoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			DefKind_ORecord: true,
		},
	},
	DefKind_WDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_WRecord: true,
		},
	},
	DefKind_GRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_GRecord: true,
		},
	},
	DefKind_CRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_CRecord: true,
		},
	},
	DefKind_ORecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_ORecord: true,
		},
	},
	DefKind_WRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_WRecord: true,
		},
	},
	DefKind_ViewRecord: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_ViewRecord_PartitionKey:      true,
			DefKind_ViewRecord_ClusteringColumns: true,
			DefKind_ViewRecord_Value:             true,
		},
		systemFields: map[string]bool{},
	},
	DefKind_ViewRecord_PartitionKey: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			DataKind_int32:    true,
			DataKind_int64:    true,
			DataKind_float32:  true,
			DataKind_float64:  true,
			DataKind_QName:    true,
			DataKind_bool:     true,
			DataKind_RecordID: true,
		},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[DefKind]bool{},
	},
	DefKind_ViewRecord_ClusteringColumns: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[DefKind]bool{},
	},
	DefKind_ViewRecord_Value: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed:       false,
		availableContainerKinds: map[DefKind]bool{},
	},
	DefKind_Object: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_Element: true,
		},
	},
	DefKind_Element: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
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
		containersAllowed: true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_Element: true,
		},
	},
	DefKind_QueryFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_GDoc:   true,
			DefKind_CDoc:   true,
			DefKind_ODoc:   true,
			DefKind_WDoc:   true,
			DefKind_Object: true,
		},
	},
	DefKind_CommandFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_GDoc:   true,
			DefKind_CDoc:   true,
			DefKind_ODoc:   true,
			DefKind_WDoc:   true,
			DefKind_Object: true,
		},
	},
}
