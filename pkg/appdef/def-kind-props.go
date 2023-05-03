/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Definition kind properties
var defKindProps = map[DefKind]struct {
	structure               bool
	fieldsAllowed           bool
	availableFieldKinds     map[DataKind]bool
	systemFields            map[string]bool
	containersAllowed       bool
	availableContainerKinds map[DefKind]bool
	availableUniques        bool
}{
	DefKind_null: {
		structure:               false,
		fieldsAllowed:           false,
		availableFieldKinds:     map[DataKind]bool{},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[DefKind]bool{},
		availableUniques:        false,
	},
	DefKind_GDoc: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_CDoc: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_ODoc: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_WDoc: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_GRecord: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_CRecord: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_ORecord: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_WRecord: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_ViewRecord: {
		structure:           false,
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[DefKind]bool{
			DefKind_ViewRecord_PartitionKey:      true,
			DefKind_ViewRecord_ClusteringColumns: true,
			DefKind_ViewRecord_Value:             true,
		},
		availableUniques: false,
	},
	DefKind_ViewRecord_PartitionKey: {
		structure:     false,
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
		availableUniques:        false,
	},
	DefKind_ViewRecord_ClusteringColumns: {
		structure:     false,
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
		availableUniques:        false,
	},
	DefKind_ViewRecord_Value: {
		structure:     false,
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
		availableUniques:        false,
	},
	DefKind_Object: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_Element: {
		structure:     true,
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
		availableUniques: true,
	},
	DefKind_QueryFunction: {
		structure:           false,
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
		availableUniques: false,
	},
	DefKind_CommandFunction: {
		structure:           false,
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
		availableUniques: false,
	},
}
