/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

func (g *schemaGenerator) generateSchema(ischema ischema, op appdef.OperationKind, fieldNames *[]appdef.FieldName) map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	// Add fields to schema
	for _, field := range ischema.Fields() {
		// Do not skip system fields in schema
		// if field.IsSys() {
		// 	continue
		// }

		if fieldNames != nil && len(*fieldNames) > 0 && !containsFieldName(*fieldNames, field.Name()) {
			continue
		}

		fieldSchema := g.generateFieldSchema(field, op)

		properties[field.Name()] = fieldSchema

		if field.Required() {
			required = append(required, field.Name())
		}
	}

	if hasContainers, ok := ischema.(appdef.IWithContainers); ok {
		for _, container := range hasContainers.Containers() {
			schemaName, ok := g.docSchemaRefIfExist(container.QName(), op)
			if !ok {
				continue // doc and/or operation not available to this role
			}
			properties[container.Name()] = map[string]interface{}{
				schemaKeyType: schemaTypeArray,
				schemaKeyItems: map[string]interface{}{
					schemaKeyRef: schemaName,
				},
				propertyMinItems: container.MinOccurs(),
				propertyMaxItems: container.MaxOccurs(),
			}
		}
	}

	schema := map[string]interface{}{
		schemaKeyType:       schemaTypeObject,
		schemaKeyProperties: properties,
	}

	if len(required) > 0 {
		schema[schemaKeyRequired] = required
	}

	if ischema.Comment() != "" {
		schema[schemaKeyDescription] = ischema.Comment()
	}

	return schema
}

// generateFieldSchema creates a schema for a specific field
func (g *schemaGenerator) generateFieldSchema(field appdef.IField, op appdef.OperationKind) map[string]interface{} {
	schema := make(map[string]interface{})

	// Handle reference fields
	if refField, isRef := field.(appdef.IRefField); isRef {

		oneOf := make([]map[string]interface{}, 0, len(refField.Refs())+1)
		oneOf = append(oneOf, map[string]interface{}{
			schemaKeyType:   schemaTypeInteger,
			schemaKeyFormat: schemaFormatInt64,
		})
		refNames := make([]string, 0, len(refField.Refs()))

		if len(refField.Refs()) > 0 && op == appdef.OperationKind_Select {
			for i := 0; i < len(refField.Refs()); i++ {
				schemaRef, ok := g.docSchemaRefIfExist(refField.Refs()[i], op)
				if !ok {
					continue // referenced document not available to this role
				}
				typeName := refField.Refs()[i].String()
				oneOf = append(oneOf, map[string]interface{}{
					schemaKeyRef: schemaRef,
				})
				refNames = append(refNames, typeName)
			}
		}

		if len(oneOf) > 1 {
			schema[schemaKeyOneOf] = oneOf
			schema[schemaKeyDescription] = fmt.Sprintf(descrIDOf, strings.Join(refNames, ", "))
		} else {
			schema[schemaKeyType] = schemaTypeInteger
			schema[schemaKeyFormat] = schemaFormatInt64
			schema[schemaKeyDescription] = descrRefToDocRecord
		}

		return schema
	}

	// Handle regular fields based on data kind
	switch field.DataKind() {
	case appdef.DataKind_int32:
		schema[schemaKeyType] = schemaTypeInteger
		schema[schemaKeyFormat] = schemaFormatInt32
	case appdef.DataKind_int64:
		schema[schemaKeyType] = schemaTypeInteger
		schema[schemaKeyFormat] = schemaFormatInt64
	case appdef.DataKind_float32:
		schema[schemaKeyType] = schemaTypeNumber
		schema[schemaKeyFormat] = schemaFormatFloat
	case appdef.DataKind_float64:
		schema[schemaKeyType] = schemaTypeNumber
		schema[schemaKeyFormat] = schemaFormatDouble
	case appdef.DataKind_bool:
		schema[schemaKeyType] = schemaTypeBoolean
	case appdef.DataKind_string:
		schema[schemaKeyType] = schemaTypeString
		// Add max length constraint if exists
		if constraint, ok := field.Constraints()[appdef.ConstraintKind_MaxLen]; ok {
			maxLen, _ := constraint.Value().(uint16)
			schema[propertyMaxLength] = maxLen
		}
	case appdef.DataKind_bytes:
		schema[schemaKeyType] = schemaTypeString
		schema[schemaKeyFormat] = schemaFormatByte
		// Add max length constraint if exists
		if constraint, ok := field.Constraints()[appdef.ConstraintKind_MaxLen]; ok {
			maxLen, _ := constraint.Value().(uint16)
			schema[propertyMaxLength] = maxLen
		}
	case appdef.DataKind_QName:
		schema[schemaKeyType] = schemaTypeString
		schema[propertyPattern] = qNamePatternRegex
		schema[propertyExample] = qNameExample
	case appdef.DataKind_RecordID:
		schema[schemaKeyType] = schemaTypeInteger
		schema[schemaKeyFormat] = schemaFormatInt64
	default:
		schema[schemaKeyType] = schemaTypeString
	}

	// Add field comment as description
	if field.Comment() != "" {
		schema[schemaKeyDescription] = field.Comment()
	}

	return schema
}

// Helper function to check if a field name is in a list of field names
func containsFieldName(fieldNames []appdef.FieldName, name appdef.FieldName) bool {
	for _, n := range fieldNames {
		if n == name {
			return true
		}
	}
	return false
}
