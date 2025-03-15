/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 */

package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
)

type SchemaMeta struct {
	SchemaTitle   string
	SchemaVersion string
}

/*
PublishedTypes lists the resources allowed to the published role in the workspace and ancestors (including resources available to non-authenticated requests):

  - Documents

  - Views

  - Commands

  - Queries

    When fieldNames is empty, it means all fields are allowed

Usage:

	for t, ops := range acl.PublishedTypes(ws, role) {
	  for op, fields := range ops {
	    if fields == nil {
	      fmt.Println(t, op, "all fields")
	    } else {
	      fmt.Println(t, op, *fields...)
	    }
	  }
	}
*/
type PublishedTypesFunc func(ws appdef.IWorkspace, role appdef.QName) iter.Seq2[appdef.IType,
	iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]

// CreateOpenApiSchema generates an OpenAPI schema document for the given workspace and role
func CreateOpenApiSchema(writer io.Writer, ws appdef.IWorkspace, role appdef.QName,
	pubTypesFunc PublishedTypesFunc, meta SchemaMeta) error {

	generator := &schemaGenerator{
		ws:             ws,
		role:           role,
		pubTypesFunc:   pubTypesFunc,
		meta:           meta,
		components:     make(map[string]interface{}),
		paths:          make(map[string]map[string]interface{}),
		schemasByType:  make(map[string]map[appdef.OperationKind]string),
		processedTypes: make(map[string]bool),
	}

	if err := generator.generate(); err != nil {
		return err
	}

	return generator.write(writer)
}

type schemaGenerator struct {
	ws             appdef.IWorkspace
	role           appdef.QName
	pubTypesFunc   PublishedTypesFunc
	meta           SchemaMeta
	components     map[string]interface{}
	paths          map[string]map[string]interface{}
	schemasByType  map[string]map[appdef.OperationKind]string
	processedTypes map[string]bool
}

// generate performs the schema generation process
func (g *schemaGenerator) generate() error {
	// First pass - generate schema components for types
	if err := g.generateComponents(); err != nil {
		return err
	}

	// Second pass - generate paths using components
	if err := g.generatePaths(); err != nil {
		return err
	}

	return nil
}

// generateComponents creates schema components for all published types
func (g *schemaGenerator) generateComponents() error {
	schemas := make(map[string]interface{})
	g.components["schemas"] = schemas

	for t, ops := range g.pubTypesFunc(g.ws, g.role) {
		typeName := fmt.Sprintf("%s.%s", t.QName().Pkg(), t.QName().Entity())

		// Skip if already processed
		if g.processedTypes[typeName] {
			continue
		}

		for op, fields := range ops {
			if err := g.generateSchemaComponent(t, op, fields, schemas); err != nil {
				return err
			}
		}

		g.processedTypes[typeName] = true
	}

	return nil
}

// generateSchemaComponent creates a schema component for a specific type and operation
func (g *schemaGenerator) generateSchemaComponent(typ appdef.IType, op appdef.OperationKind,
	fieldNames *[]appdef.FieldName, schemas map[string]interface{}) error {

	typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	// If no field constraints (all fields allowed) or fieldNames is nil
	useAllFields := fieldNames == nil || len(*fieldNames) == 0

	// Create a component schema name based on type and operation if fields are restricted
	componentName := typeName
	if !useAllFields {
		componentName = fmt.Sprintf("%s_%s", typeName, op.String())
	}

	// Save the schema reference for this type and operation
	if _, ok := g.schemasByType[typeName]; !ok {
		g.schemasByType[typeName] = make(map[appdef.OperationKind]string)
	}
	g.schemasByType[typeName][op] = componentName

	// Create the schema component
	withFields, ok := typ.(appdef.IWithFields)
	if !ok {
		return nil // Type doesn't have fields, skip
	}

	properties := make(map[string]interface{})
	required := make([]string, 0)

	// Add fields to schema
	for _, field := range withFields.Fields() {
		// Skip system fields in schema
		if field.IsSys() {
			continue
		}

		// Check if this field is allowed for this operation
		if !useAllFields && !containsFieldName(*fieldNames, field.Name()) {
			continue
		}

		fieldSchema := g.generateFieldSchema(field)

		properties[field.Name()] = fieldSchema

		if field.Required() {
			required = append(required, field.Name())
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	if typ.Comment() != "" {
		schema["description"] = typ.Comment()
	}

	schemas[componentName] = schema

	return nil
}

// generateFieldSchema creates a schema for a specific field
func (g *schemaGenerator) generateFieldSchema(field appdef.IField) map[string]interface{} {
	schema := make(map[string]interface{})

	// Handle reference fields
	if refField, isRef := field.(appdef.IRefField); isRef {
		schema["type"] = "integer"
		schema["format"] = "int64"

		if len(refField.Refs()) > 0 {
			refNames := make([]string, 0, len(refField.Refs()))
			for _, ref := range refField.Refs() {
				refNames = append(refNames, fmt.Sprintf("%s.%s", ref.Pkg(), ref.Entity()))
			}
			schema["description"] = fmt.Sprintf("Reference to: %s", strings.Join(refNames, ", "))
		} else {
			schema["description"] = "Reference to any document/record"
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

// generatePaths creates path items for all published types and their operations
func (g *schemaGenerator) generatePaths() error {
	for t, ops := range g.pubTypesFunc(g.ws, g.role) {
		for op, _ := range ops {
			path, method, err := g.getPathAndMethod(t, op)
			if err != nil {
				continue // Skip operations we cannot map to paths
			}

			if err := g.addPathItem(path, method, t, op); err != nil {
				return err
			}
		}
	}

	return nil
}

// getPathAndMethod returns the API path and HTTP method for a given type and operation
func (g *schemaGenerator) getPathAndMethod(typ appdef.IType, op appdef.OperationKind) (string, string, error) {
	typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	switch typ.Kind() {
	case appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		switch op {
		case appdef.OperationKind_Insert:
			return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/%s", typeName), "post", nil
		case appdef.OperationKind_Update:
			return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/%s/{id}", typeName), "patch", nil
		case appdef.OperationKind_Deactivate:
			return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/%s/{id}", typeName), "delete", nil
		case appdef.OperationKind_Select:
			return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/%s/{id}", typeName), "get", nil
		}
	}

	// Special handling for CDoc collection reading
	if typ.Kind() == appdef.TypeKind_CDoc && op == appdef.OperationKind_Select {
		return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/cdocs/%s", typeName), "get", nil
	}

	// Special handling for commands
	if _, ok := typ.(appdef.ICommand); ok && op == appdef.OperationKind_Execute {
		return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/commands/%s", typeName), "post", nil
	}

	// Special handling for queries
	if _, ok := typ.(appdef.IQuery); ok && op == appdef.OperationKind_Execute {
		return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/queries/%s", typeName), "get", nil
	}

	// Special handling for views
	if _, ok := typ.(appdef.IView); ok && op == appdef.OperationKind_Select {
		return fmt.Sprintf("/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/views/%s", typeName), "get", nil
	}

	return "", "", fmt.Errorf("unsupported type %s for operation %s", typ.Kind().String(), op.String())
}

// addPathItem adds a path item to the OpenAPI schema
func (g *schemaGenerator) addPathItem(path, method string, typ appdef.IType, op appdef.OperationKind) error {
	//typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	// Create path if it doesn't exist
	if _, exists := g.paths[path]; !exists {
		g.paths[path] = make(map[string]interface{})
	}

	// Create operation object
	operation := make(map[string]interface{})

	// Add tags based on type's tags
	tags := g.generateTags(typ)
	if len(tags) > 0 {
		operation["tags"] = tags
	}

	// Add operation description
	operation["description"] = g.generateDescription(typ, op)

	// Add operation parameters
	parameters := g.generateParameters(path)
	if len(parameters) > 0 {
		operation["parameters"] = parameters
	}

	// Add request body for appropriate methods
	if method == "post" || method == "patch" || method == "put" {
		requestBody := g.generateRequestBody(typ, op)
		if requestBody != nil {
			operation["requestBody"] = requestBody
		}
	}

	// Add responses
	operation["responses"] = g.generateResponses(typ, op)

	// Add the operation to the path
	g.paths[path][method] = operation

	return nil
}

// generateTags creates tags for a specific type
func (g *schemaGenerator) generateTags(typ appdef.IType) []string {
	tags := make([]string, 0)

	// Check if type has tags with feature values
	for _, tag := range typ.Tags() {
		if feature := tag.Feature(); feature != "" {
			tags = append(tags, feature)
		}
	}

	// If no feature tags found, use package name as default tag
	if len(tags) == 0 {
		tags = append(tags, typ.QName().Pkg())
	}

	return tags
}

// generateDescription creates description for an operation on a type
func (g *schemaGenerator) generateDescription(typ appdef.IType, op appdef.OperationKind) string {
	// Use type's comment if available
	if typ.Comment() != "" {
		return typ.Comment()
	}

	typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	// Generate default description based on type and operation
	switch {
	case typ.Kind() == appdef.TypeKind_Command || typ.Kind() == appdef.TypeKind_Query:
		if op == appdef.OperationKind_Execute {
			if typ.Kind() == appdef.TypeKind_Command {
				return fmt.Sprintf("Executes %s", typeName)
			} else {
				return fmt.Sprintf("Selects from query %s", typeName)
			}
		}

	case typ.Kind() == appdef.TypeKind_ViewRecord:
		return fmt.Sprintf("Selects from view %s", typeName)

	case typ.Kind() == appdef.TypeKind_CDoc:
		if op == appdef.OperationKind_Select {
			return fmt.Sprintf("Returns a collection of %s", typeName)
		}
	}

	// Handle document/record operations
	if appdef.TypeKind_Records.Contains(typ.Kind()) {
		switch op {
		case appdef.OperationKind_Insert:
			if appdef.TypeKind_Docs.Contains(typ.Kind()) {
				return fmt.Sprintf("Creates document %s", typeName)
			}
			return fmt.Sprintf("Creates record %s", typeName)
		case appdef.OperationKind_Update:
			if appdef.TypeKind_Docs.Contains(typ.Kind()) {
				return fmt.Sprintf("Updates document %s", typeName)
			}
			return fmt.Sprintf("Updates record %s", typeName)
		case appdef.OperationKind_Deactivate:
			if appdef.TypeKind_Docs.Contains(typ.Kind()) {
				return fmt.Sprintf("Deactivates document %s", typeName)
			}
			return fmt.Sprintf("Deactivates record %s", typeName)
		case appdef.OperationKind_Select:
			if appdef.TypeKind_Docs.Contains(typ.Kind()) {
				return fmt.Sprintf("Reads document %s", typeName)
			}
			return fmt.Sprintf("Reads record %s", typeName)
		}
	}

	return fmt.Sprintf("Operation %s on %s", op.String(), typeName)
}

// generateParameters creates parameters for a path
func (g *schemaGenerator) generateParameters(path string) []map[string]interface{} {
	parameters := make([]map[string]interface{}, 0)

	// Common parameters for all paths
	if strings.Contains(path, "{owner}") {
		parameters = append(parameters, map[string]interface{}{
			"name":     "owner",
			"in":       "path",
			"required": true,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Name of a user who owns the application",
		})
	}

	if strings.Contains(path, "{app}") {
		parameters = append(parameters, map[string]interface{}{
			"name":     "app",
			"in":       "path",
			"required": true,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Name of an application",
		})
	}

	if strings.Contains(path, "{wsid}") {
		parameters = append(parameters, map[string]interface{}{
			"name":     "wsid",
			"in":       "path",
			"required": true,
			"schema": map[string]interface{}{
				"type":   "integer",
				"format": "int64",
			},
			"description": "The ID of workspace",
		})
	}

	if strings.Contains(path, "{id}") {
		parameters = append(parameters, map[string]interface{}{
			"name":     "id",
			"in":       "path",
			"required": true,
			"schema": map[string]interface{}{
				"type":   "integer",
				"format": "int64",
			},
			"description": "ID of a document or record",
		})
	}

	// Add query parameters for GET methods
	if strings.Contains(path, "/views/") || strings.Contains(path, "/queries/") || strings.Contains(path, "/cdocs/") {
		// Add query constraints parameters
		parameters = append(parameters, map[string]interface{}{
			"name":     "where",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Filter criteria in JSON format",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "order",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Field to order results by",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "limit",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "integer",
			},
			"description": "Maximum number of results to return",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "skip",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "integer",
			},
			"description": "Number of results to skip",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "include",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Referenced objects to include in response",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "keys",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Specific fields to include in response",
		})
	}

	// Add arg parameter for queries
	if strings.Contains(path, "/queries/") {
		parameters = append(parameters, map[string]interface{}{
			"name":     "arg",
			"in":       "query",
			"required": false,
			"schema": map[string]interface{}{
				"type": "string",
			},
			"description": "Query argument in JSON format",
		})
	}

	return parameters
}

// generateRequestBody creates a request body for a type and operation
func (g *schemaGenerator) generateRequestBody(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	// Get the schema for this type and operation
	schemaName := typeName
	if typeSchemas, ok := g.schemasByType[typeName]; ok {
		if opSchema, ok := typeSchemas[op]; ok {
			schemaName = opSchema
		}
	}

	return map[string]interface{}{
		"required": true,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/schemas/" + schemaName,
				},
			},
		},
	}
}

// generateResponses creates response objects for a type and operation
func (g *schemaGenerator) generateResponses(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	responses := make(map[string]interface{})
	typeName := fmt.Sprintf("%s.%s", typ.QName().Pkg(), typ.QName().Entity())

	// Add standard error responses
	responses["401"] = map[string]interface{}{
		"description": "Unauthorized",
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type": "string",
						},
						"status": map[string]interface{}{
							"type": "integer",
						},
						"qname": map[string]interface{}{
							"type": "string",
						},
						"data": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"message"},
				},
			},
		},
	}

	responses["403"] = map[string]interface{}{
		"description": "Forbidden",
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/responses/401/content/application~1json/schema",
				},
			},
		},
	}

	responses["404"] = map[string]interface{}{
		"description": "Not Found",
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/responses/401/content/application~1json/schema",
				},
			},
		},
	}

	// Add specific successful response based on type and operation
	switch {
	case op == appdef.OperationKind_Insert:
		responses["201"] = map[string]interface{}{
			"description": "Created",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								"type": "integer",
							},
							"NewIDs": map[string]interface{}{
								"type": "object",
								"additionalProperties": map[string]interface{}{
									"type":   "integer",
									"format": "int64",
								},
							},
						},
					},
				},
			},
		}

		// Add bad request response for create operations
		responses["400"] = map[string]interface{}{
			"description": "Bad Request",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/responses/401/content/application~1json/schema",
					},
				},
			},
		}

	case op == appdef.OperationKind_Update || op == appdef.OperationKind_Deactivate:
		responses["200"] = map[string]interface{}{
			"description": "OK",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								"type": "integer",
							},
						},
					},
				},
			},
		}

	case op == appdef.OperationKind_Execute && typ.Kind() == appdef.TypeKind_Command:
		responses["200"] = map[string]interface{}{
			"description": "OK",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								"type": "integer",
							},
						},
					},
				},
			},
		}

		// Add bad request response for command execution
		responses["400"] = map[string]interface{}{
			"description": "Bad Request",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/responses/401/content/application~1json/schema",
					},
				},
			},
		}

	case op == appdef.OperationKind_Select && appdef.TypeKind_Records.Contains(typ.Kind()):
		// Get schema reference for this type
		schemaRef := "#/components/schemas/" + typeName
		if typeSchemas, ok := g.schemasByType[typeName]; ok {
			if opSchema, ok := typeSchemas[op]; ok {
				schemaRef = "#/components/schemas/" + opSchema
			}
		}

		responses["200"] = map[string]interface{}{
			"description": "OK",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": schemaRef,
					},
				},
			},
		}

	case (op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_CDoc) ||
		(op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_ViewRecord) ||
		(op == appdef.OperationKind_Execute && typ.Kind() == appdef.TypeKind_Query):
		// Collection response with results array
		schemaRef := "#/components/schemas/" + typeName
		if typeSchemas, ok := g.schemasByType[typeName]; ok {
			if opSchema, ok := typeSchemas[op]; ok {
				schemaRef = "#/components/schemas/" + opSchema
			}
		}

		responses["200"] = map[string]interface{}{
			"description": "OK",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"results": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{
									"$ref": schemaRef,
								},
							},
							"error": map[string]interface{}{
								"$ref": "#/components/responses/401/content/application~1json/schema",
							},
						},
					},
				},
			},
		}
	}

	return responses
}

// write generates the final OpenAPI document and writes it to the provided writer
func (g *schemaGenerator) write(writer io.Writer) error {
	// Create the OpenAPI schema
	schema := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   g.meta.SchemaTitle,
			"version": g.meta.SchemaVersion,
		},
		"paths":      g.paths,
		"components": g.components,
	}

	// Serialize to JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schema)
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
