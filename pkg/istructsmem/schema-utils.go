/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

// schemaSystemFieldsRequireType: schema system fields requirements
type schemaSystemFieldsRequireType struct {
	qName, id, container, parentID, isActive bool
}

// schemaPropsType: schema properties
type schemaPropsType struct {
	fieldsAllowed     bool
	fieldsKinds       map[istructs.DataKindType]bool
	fieldsSystem      schemaSystemFieldsRequireType
	containersAllowed bool
	containersKinds   map[istructs.SchemaKindType]bool
}

// schemaPropsTypeList: type of schemas properties by kind
type schemaPropsTypeList map[istructs.SchemaKindType]schemaPropsType

var schemaProps schemaPropsTypeList = schemaPropsTypeList{
	istructs.SchemaKind_null: schemaPropsType{
		fieldsAllowed:     false,
		fieldsKinds:       map[istructs.DataKindType]bool{},
		fieldsSystem:      schemaSystemFieldsRequireType{},
		containersAllowed: false,
		containersKinds:   map[istructs.SchemaKindType]bool{},
	},
	istructs.SchemaKind_GDoc: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_GRecord: true,
		},
	},
	istructs.SchemaKind_CDoc: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_CRecord: true,
		},
	},
	istructs.SchemaKind_ODoc: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_ODoc:    true, // #19322!: ODocs should be able to contain ODocs
			istructs.SchemaKind_ORecord: true,
		},
	},
	istructs.SchemaKind_WDoc: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_WRecord: true,
		},
	},
	istructs.SchemaKind_GRecord: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, parentID: true, container: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_GRecord: true,
		},
	},
	istructs.SchemaKind_CRecord: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, parentID: true, container: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_CRecord: true,
		},
	},
	istructs.SchemaKind_ORecord: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, parentID: true, container: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_ORecord: true,
		},
	},
	istructs.SchemaKind_WRecord: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{id: true, qName: true, parentID: true, container: true, isActive: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_WRecord: true,
		},
	},
	istructs.SchemaKind_ViewRecord: schemaPropsType{
		fieldsAllowed:     false,
		fieldsKinds:       map[istructs.DataKindType]bool{},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_ViewRecord_PartitionKey:      true,
			istructs.SchemaKind_ViewRecord_ClusteringColumns: true,
			istructs.SchemaKind_ViewRecord_Value:             true,
		},
		fieldsSystem: schemaSystemFieldsRequireType{},
	},
	istructs.SchemaKind_ViewRecord_PartitionKey: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
			istructs.DataKind_int32:    true,
			istructs.DataKind_int64:    true,
			istructs.DataKind_float32:  true,
			istructs.DataKind_float64:  true,
			istructs.DataKind_QName:    true,
			istructs.DataKind_bool:     true,
			istructs.DataKind_RecordID: true,
		},
		fieldsSystem:      schemaSystemFieldsRequireType{},
		containersAllowed: false,
		containersKinds:   map[istructs.SchemaKindType]bool{},
	},
	istructs.SchemaKind_ViewRecord_ClusteringColumns: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{},
		containersAllowed: false,
		containersKinds:   map[istructs.SchemaKindType]bool{},
	},
	istructs.SchemaKind_ViewRecord_Value: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{qName: true},
		containersAllowed: false,
		containersKinds:   map[istructs.SchemaKindType]bool{},
	},
	istructs.SchemaKind_Object: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{qName: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_Element: true,
		},
	},
	istructs.SchemaKind_Element: schemaPropsType{
		fieldsAllowed: true,
		fieldsKinds: map[istructs.DataKindType]bool{
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
		fieldsSystem:      schemaSystemFieldsRequireType{qName: true, container: true},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_Element: true,
		},
	},
	istructs.SchemaKind_QueryFunction: schemaPropsType{
		fieldsAllowed:     false,
		fieldsKinds:       map[istructs.DataKindType]bool{},
		fieldsSystem:      schemaSystemFieldsRequireType{},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_GDoc:   true,
			istructs.SchemaKind_CDoc:   true,
			istructs.SchemaKind_ODoc:   true,
			istructs.SchemaKind_WDoc:   true,
			istructs.SchemaKind_Object: true,
		},
	},
	istructs.SchemaKind_CommandFunction: schemaPropsType{
		fieldsAllowed:     false,
		fieldsKinds:       map[istructs.DataKindType]bool{},
		fieldsSystem:      schemaSystemFieldsRequireType{},
		containersAllowed: true,
		containersKinds: map[istructs.SchemaKindType]bool{
			istructs.SchemaKind_GDoc:   true,
			istructs.SchemaKind_CDoc:   true,
			istructs.SchemaKind_ODoc:   true,
			istructs.SchemaKind_WDoc:   true,
			istructs.SchemaKind_Object: true,
		},
	},
}

// availableFieldKind: is field kind available in schema kind
func availableFieldKind(schKind istructs.SchemaKindType, fieldKind istructs.DataKindType) bool {
	return schemaProps[schKind].fieldsKinds[fieldKind]
}

// availableContainers: is schema kind can contains nested containers
func availableContainers(kind istructs.SchemaKindType) bool {
	return schemaProps[kind].containersAllowed
}

// availableContainerKind: is schema kind can contains nested containers
func availableContainerKind(kind, contKind istructs.SchemaKindType) bool {
	return schemaProps[kind].containersKinds[contKind]
}

// schemaNeedSysField_QName: is schema kind need sys.QName field
func schemaNeedSysField_QName(kind istructs.SchemaKindType) bool {
	if !schemaProps[kind].fieldsAllowed {
		return false
	}
	return schemaProps[kind].fieldsSystem.qName
}

// schemaNeedSysField_ID: is schema kind need sys.ID field
func schemaNeedSysField_ID(kind istructs.SchemaKindType) bool {
	if !schemaProps[kind].fieldsAllowed {
		return false
	}
	return schemaProps[kind].fieldsSystem.id
}

// schemaNeedSysField_ParentID: is schema kind need sys.ParentID field
func schemaNeedSysField_ParentID(kind istructs.SchemaKindType) bool {
	if !schemaProps[kind].fieldsAllowed {
		return false
	}
	return schemaProps[kind].fieldsSystem.parentID
}

// schemaNeedSysField_Container: is schema kind need sys.Container field
func schemaNeedSysField_Container(kind istructs.SchemaKindType) bool {
	if !schemaProps[kind].fieldsAllowed {
		return false
	}
	return schemaProps[kind].fieldsSystem.container
}

// schemaNeedSysField_IsActive: is schema kind need sys.IsActive field
func schemaNeedSysField_IsActive(kind istructs.SchemaKindType) bool {
	if !schemaProps[kind].fieldsAllowed {
		return false
	}
	return schemaProps[kind].fieldsSystem.isActive
}

// schemaNeedSysFieldMask returns system fields mask combination for schema kind, see sfm_××× consts
func schemaNeedSysFieldMask(kind istructs.SchemaKindType) uint16 {
	sfm := uint16(0)
	if schemaNeedSysField_ID(kind) {
		sfm |= sfm_ID
	}
	if schemaNeedSysField_ParentID(kind) {
		sfm |= sfm_ParentID
	}
	if schemaNeedSysField_Container(kind) {
		sfm |= sfm_Container
	}
	if schemaNeedSysField_IsActive(kind) {
		sfm |= sfm_IsActive
	}
	return sfm
}

func validateOccurs(min, max istructs.ContainerOccursType) error {
	if max == 0 {
		return ErrMaxOccursMissed
	}
	if max < min {
		return ErrMaxOccursLessMinOccurs
	}
	return nil
}

type validateErrorType struct {
	error
	code int
}

func (e validateErrorType) Code() int {
	return e.code
}

func (e validateErrorType) Unwrap() error {
	return e.error
}

func validateError(code int, err error) ValidateError {
	e := validateErrorType{
		error: fmt.Errorf("%w; validate error code: %d", err, code),
		code:  code,
	}
	return e
}

func validateErrorf(code int, format string, a ...interface{}) ValidateError {
	return validateError(code, fmt.Errorf(format, a...))
}
