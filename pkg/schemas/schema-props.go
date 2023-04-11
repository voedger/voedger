/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"github.com/untillpro/voedger/pkg/istructs"
)

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
	istructs.SchemaKind_null: {
		fieldsAllowed:           false,
		availableFieldKinds:     map[DataKind]bool{},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[SchemaKind]bool{},
	},
	istructs.SchemaKind_GDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:       true,
			istructs.SystemField_QName:    true,
			istructs.SystemField_IsActive: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_GRecord: true,
		},
	},
	istructs.SchemaKind_CDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:       true,
			istructs.SystemField_QName:    true,
			istructs.SystemField_IsActive: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_CRecord: true,
		},
	},
	istructs.SchemaKind_ODoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:    true,
			istructs.SystemField_QName: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			istructs.SchemaKind_ORecord: true,
		},
	},
	istructs.SchemaKind_WDoc: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:       true,
			istructs.SystemField_QName:    true,
			istructs.SystemField_IsActive: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_WRecord: true,
		},
	},
	istructs.SchemaKind_GRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:        true,
			istructs.SystemField_QName:     true,
			istructs.SystemField_ParentID:  true,
			istructs.SystemField_Container: true,
			istructs.SystemField_IsActive:  true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_GRecord: true,
		},
	},
	istructs.SchemaKind_CRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:        true,
			istructs.SystemField_QName:     true,
			istructs.SystemField_ParentID:  true,
			istructs.SystemField_Container: true,
			istructs.SystemField_IsActive:  true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_CRecord: true,
		},
	},
	istructs.SchemaKind_ORecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:        true,
			istructs.SystemField_QName:     true,
			istructs.SystemField_ParentID:  true,
			istructs.SystemField_Container: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_ORecord: true,
		},
	},
	istructs.SchemaKind_WRecord: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_ID:        true,
			istructs.SystemField_QName:     true,
			istructs.SystemField_ParentID:  true,
			istructs.SystemField_Container: true,
			istructs.SystemField_IsActive:  true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_WRecord: true,
		},
	},
	istructs.SchemaKind_ViewRecord: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_ViewRecord_PartitionKey:      true,
			istructs.SchemaKind_ViewRecord_ClusteringColumns: true,
			istructs.SchemaKind_ViewRecord_Value:             true,
		},
		systemFields: map[string]bool{},
	},
	istructs.SchemaKind_ViewRecord_PartitionKey: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[SchemaKind]bool{},
	},
	istructs.SchemaKind_ViewRecord_ClusteringColumns: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true, // last field
			istructs.DataKind_string:   true, // last field
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields:            map[string]bool{},
		containersAllowed:       false,
		availableContainerKinds: map[SchemaKind]bool{},
	},
	istructs.SchemaKind_ViewRecord_Value: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
			istructs.DataKind_Record:   true, // +
			istructs.DataKind_Event:    true, // +
		},
		systemFields: map[string]bool{
			istructs.SystemField_QName: true,
		},
		containersAllowed:       false,
		availableContainerKinds: map[SchemaKind]bool{},
	},
	istructs.SchemaKind_Object: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_QName: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_Element: true,
		},
	},
	istructs.SchemaKind_Element: {
		fieldsAllowed: true,
		availableFieldKinds: map[DataKind]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_bytes:    true,
			istructs.DataKind_string:   true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		systemFields: map[string]bool{
			istructs.SystemField_QName:     true,
			istructs.SystemField_Container: true,
		},
		containersAllowed: true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_Element: true,
		},
	},
	istructs.SchemaKind_QueryFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_GDoc:   true,
			istructs.SchemaKind_CDoc:   true,
			istructs.SchemaKind_ODoc:   true,
			istructs.SchemaKind_WDoc:   true,
			istructs.SchemaKind_Object: true,
		},
	},
	istructs.SchemaKind_CommandFunction: {
		fieldsAllowed:       false,
		availableFieldKinds: map[DataKind]bool{},
		systemFields:        map[string]bool{},
		containersAllowed:   true,
		availableContainerKinds: map[SchemaKind]bool{
			istructs.SchemaKind_GDoc:   true,
			istructs.SchemaKind_CDoc:   true,
			istructs.SchemaKind_ODoc:   true,
			istructs.SchemaKind_WDoc:   true,
			istructs.SchemaKind_Object: true,
		},
	},
}
