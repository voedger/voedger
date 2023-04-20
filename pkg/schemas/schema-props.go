/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

// Schema kind properties
type SchemaKindProps struct {
	fieldsAllowed           bool
	availableFieldKinds     map[DataKind]bool
	systemFields            map[string]bool
	containersAllowed       bool
	availableContainerKinds map[SchemaKind]bool
}

// Is fields allowed.
func (props SchemaKindProps) FieldsAllowed() bool {
	return props.fieldsAllowed
}

// Is data kind allowed.
func (props SchemaKindProps) DataKindAvailable(k DataKind) bool {
	return props.fieldsAllowed && props.availableFieldKinds[k]
}

// Is specified system field used.
func (props SchemaKindProps) HasSystemField(f string) bool {
	return props.fieldsAllowed && props.systemFields[f]
}

// Is containers allowed.
func (props SchemaKindProps) ContainersAllowed() bool {
	return props.containersAllowed
}

// Is specified schema kind may be used in child containers.
func (props SchemaKindProps) ContainerKindAvailable(k SchemaKind) bool {
	return props.containersAllowed && props.availableContainerKinds[k]
}

var schemaKindProps = map[SchemaKind]SchemaKindProps{
	SchemaKind_null: {
		fieldsAllowed:           false,
		availableFieldKinds:     map[DataKind]bool{},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[SchemaKind]bool{},
	},
	SchemaKind_GDoc: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_GRecord: true,
		},
	},
	SchemaKind_CDoc: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_CRecord: true,
		},
	},
	SchemaKind_ODoc: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			SchemaKind_ORecord: true,
		},
	},
	SchemaKind_WDoc: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_WRecord: true,
		},
	},
	SchemaKind_GRecord: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_GRecord: true,
		},
	},
	SchemaKind_CRecord: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_CRecord: true,
		},
	},
	SchemaKind_ORecord: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_ORecord: true,
		},
	},
	SchemaKind_WRecord: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_WRecord: true,
		},
	},
	SchemaKind_ViewRecord: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_ViewRecord_PartitionKey:      true,
			SchemaKind_ViewRecord_ClusteringColumns: true,
			SchemaKind_ViewRecord_Value:             true,
		},
		systemFields: map[string]bool{},
	},
	SchemaKind_ViewRecord_PartitionKey: {
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
		availableContainerKinds: map[SchemaKind]bool{},
	},
	SchemaKind_ViewRecord_ClusteringColumns: {
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
		availableContainerKinds: map[SchemaKind]bool{},
	},
	SchemaKind_ViewRecord_Value: {
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
		availableContainerKinds: map[SchemaKind]bool{},
	},
	SchemaKind_Object: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_Element: true,
		},
	},
	SchemaKind_Element: {
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
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_Element: true,
		},
	},
	SchemaKind_QueryFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_GDoc:   true,
			SchemaKind_CDoc:   true,
			SchemaKind_ODoc:   true,
			SchemaKind_WDoc:   true,
			SchemaKind_Object: true,
		},
	},
	SchemaKind_CommandFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			SchemaKind_GDoc:   true,
			SchemaKind_CDoc:   true,
			SchemaKind_ODoc:   true,
			SchemaKind_WDoc:   true,
			SchemaKind_Object: true,
		},
	},
}
