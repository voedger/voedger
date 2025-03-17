/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package openapi

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

	if hasContainers := ischema.(appdef.IWithContainers); hasContainers != nil {
		for _, container := range hasContainers.Containers() {
			if _, ok := g.docTypes[container.QName()]; !ok {
				continue // container not available to this role
			}
			properties[container.Name()] = map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", g.schemaNameByTypeName(container.QName().String(), op)),
				},
				"minItems": container.MinOccurs(),
				"maxItems": container.MaxOccurs(),
			}
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	if ischema.Comment() != "" {
		schema["description"] = ischema.Comment()
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
			"type":   "integer",
			"format": "int64",
		})
		refNames := make([]string, 0, len(refField.Refs()))

		if len(refField.Refs()) > 0 {
			for i := 0; i < len(refField.Refs()); i++ {
				if _, ok := g.docTypes[refField.Refs()[i]]; !ok {
					continue // referenced document not available to this role
				}
				typeName := refField.Refs()[i].String()
				oneOf = append(oneOf, map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", g.schemaNameByTypeName(typeName, op)),
				})
				refNames = append(refNames, typeName)
			}
		}

		if len(oneOf) > 1 {
			schema["oneOf"] = oneOf
			schema["description"] = fmt.Sprintf("ID of: %s", strings.Join(refNames, ", "))
		} else {
			schema["type"] = "integer"
			schema["format"] = "int64"
			schema["description"] = "Reference to a document or record"
		}

		return schema
	}

	// Handle regular fields based on data kind
	switch field.DataKind() {
	case appdef.DataKind_int32:
		schema["type"] = "integer"
		schema["format"] = "int32"
	case appdef.DataKind_int64:
		schema["type"] = "integer"
		schema["format"] = "int64"
	case appdef.DataKind_float32:
		schema["type"] = "number"
		schema["format"] = "float"
	case appdef.DataKind_float64:
		schema["type"] = "number"
		schema["format"] = "double"
	case appdef.DataKind_bool:
		schema["type"] = "boolean"
	case appdef.DataKind_string:
		schema["type"] = "string"
		// Add max length constraint if exists
		if constraint, ok := field.Constraints()[appdef.ConstraintKind_MaxLen]; ok {
			maxLen, _ := constraint.Value().(uint16)
			schema["maxLength"] = maxLen
		}
	case appdef.DataKind_bytes:
		schema["type"] = "string"
		schema["format"] = "byte"
		// Add max length constraint if exists
		if constraint, ok := field.Constraints()[appdef.ConstraintKind_MaxLen]; ok {
			maxLen, _ := constraint.Value().(uint16)
			schema["maxLength"] = maxLen
		}
	case appdef.DataKind_QName:
		schema["type"] = "string"
		schema["pattern"] = "^[a-zA-Z0-9_]+\\.[a-zA-Z0-9_]+$"
		schema["example"] = "app1pkg.MyType"
	case appdef.DataKind_RecordID:
		schema["type"] = "integer"
		schema["format"] = "int64"
	default:
		schema["type"] = "string"
	}

	// Add field comment as description
	if field.Comment() != "" {
		schema["description"] = field.Comment()
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
