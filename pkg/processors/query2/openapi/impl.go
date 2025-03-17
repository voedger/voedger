/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/processors/query2"
)

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
	ws           appdef.IWorkspace
	role         appdef.QName
	pubTypesFunc PublishedTypesFunc
	meta         SchemaMeta
	types        iter.Seq2[appdef.IType,
		iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]
	components     map[string]interface{}
	paths          map[string]map[string]interface{}
	schemasByType  map[string]map[appdef.OperationKind]string
	docTypes       map[appdef.QName]bool
	processedTypes map[string]bool
}

// generate performs the schema generation process
func (g *schemaGenerator) generate() error {
	g.types = g.pubTypesFunc(g.ws, g.role)
	g.collectDocSchemaTypes()

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

	for t, ops := range g.types {
		typeName := t.QName().String()

		// Skip if already processed
		if g.processedTypes[typeName] {
			continue
		}

		for op, fields := range ops {
			if err := g.generateSchemaComponent(t, op, fields, schemas); err != nil {
				return err
			}
			if t.Kind() == appdef.TypeKind_Command && op == appdef.OperationKind_Execute {
				// If command param is an ODoc, generate a schema for it
				cmd := t.(appdef.ICommand)
				param := cmd.Param()
				if _, ok := param.(appdef.IODoc); ok {
					if err := g.generateSchemaComponent(param.(ischema), op, nil, schemas); err != nil {
						return err
					}
				}
				if result := cmd.Result(); result != nil {
					if err := g.generateSchemaComponent(result.(ischema), op, nil, schemas); err != nil {
						return err
					}
				}
			}
			if t.Kind() == appdef.TypeKind_Query && op == appdef.OperationKind_Execute {
				qry := t.(appdef.IQuery)
				if result := qry.Result(); result != nil {
					if err := g.generateSchemaComponent(result.(ischema), op, nil, schemas); err != nil {
						return err
					}
				}
			}
		}

		g.processedTypes[typeName] = true
	}

	// generate error schema
	schemas[errorSchemaName] = map[string]interface{}{
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
	}

	return nil
}

func (g *schemaGenerator) collectDocSchemaTypes() {
	g.docTypes = make(map[appdef.QName]bool)
	for t := range g.types {
		if appdef.TypeKind_Docs.Contains(t.Kind()) {
			g.docTypes[t.QName()] = true
		}
	}
}

// generateSchemaComponent creates a schema component for a specific type and operation
func (g *schemaGenerator) generateSchemaComponent(typ appdef.IType, op appdef.OperationKind,
	fieldNames *[]appdef.FieldName, schemas map[string]interface{}) error {

	typeName := typ.QName().String()

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
	withFields, ok := typ.(ischema)
	if !ok {
		return nil // Type doesn't have fields, skip
	}

	schemas[componentName] = g.generateSchema(withFields, op, fieldNames)

	if typ.Kind() == appdef.TypeKind_ODoc || typ.Kind() == appdef.TypeKind_ORecord {
		// Generate schema components ODoc inner containers
		withContainers := typ.(appdef.IWithContainers)
		for _, container := range withContainers.Containers() {
			err := g.generateSchemaComponent(container.Type(), op, nil, schemas)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// generatePaths creates path items for all published types and their operations
func (g *schemaGenerator) generatePaths() error {
	for t, ops := range g.types {
		for op := range ops {
			paths := g.getPaths(t, op)
			for _, path := range paths {
				if err := g.addPathItem(path.Path, path.Method, t, op, path.ApiPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getPathAndMethod returns the API path and HTTP method for a given type and operation
func (g *schemaGenerator) getPaths(typ appdef.IType, op appdef.OperationKind) []pathItem {
	typeName := typ.QName().String()

	owner := "{owner}"
	app := "{app}"
	if g.meta.AppName != appdef.NullAppQName {
		owner = g.meta.AppName.Owner()
		app = g.meta.AppName.Name()
	}

	switch typ.Kind() {
	case appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		switch op {
		case appdef.OperationKind_Insert:
			return []pathItem{
				{
					Method:  methodPost,
					Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/docs/%s", owner, app, typeName),
					ApiPath: query2.ApiPath_Docs,
				},
			}
		case appdef.OperationKind_Update:
			return []pathItem{
				{
					Method:  methodPatch,
					Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/docs/%s/{id}", owner, app, typeName),
					ApiPath: query2.ApiPath_Docs,
				},
			}
		case appdef.OperationKind_Deactivate:
			return []pathItem{
				{
					Method:  methodDelete,
					Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/docs/%s/{id}", owner, app, typeName),
					ApiPath: query2.ApiPath_Docs,
				},
			}
		case appdef.OperationKind_Select:
			return []pathItem{
				{
					Method:  methodGet,
					Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/docs/%s/{id}", owner, app, typeName),
					ApiPath: query2.ApiPath_Docs,
				},
				{
					Method:  methodGet,
					Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/cdocs/%s", owner, app, typeName),
					ApiPath: query2.ApiPath_CDocs,
				},
			}

		}
	}

	if _, ok := typ.(appdef.ICommand); ok && op == appdef.OperationKind_Execute {
		return []pathItem{
			{
				Method:  methodPost,
				Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/commands/%s", owner, app, typeName),
				ApiPath: query2.ApiPath_Commands,
			},
		}
	}

	if _, ok := typ.(appdef.IQuery); ok && op == appdef.OperationKind_Execute {
		return []pathItem{
			{
				Method:  methodGet,
				Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/queries/%s", owner, app, typeName),
				ApiPath: query2.ApiPath_Queries,
			},
		}
	}

	if _, ok := typ.(appdef.IView); ok && op == appdef.OperationKind_Select {
		return []pathItem{
			{
				Method:  methodGet,
				Path:    fmt.Sprintf("/api/v2/users/%s/apps/%s/workspaces/{wsid}/views/%s", owner, app, typeName),
				ApiPath: query2.ApiPath_Views,
			},
		}
	}

	return nil
}

// addPathItem adds a path item to the OpenAPI schema
func (g *schemaGenerator) addPathItem(path, method string, typ appdef.IType, op appdef.OperationKind, apiPath query2.ApiPath) error {
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
	operation["description"] = g.generateDescription(typ, op, apiPath)

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
func (g *schemaGenerator) generateDescription(typ appdef.IType, op appdef.OperationKind, apiPath query2.ApiPath) string {
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
			if apiPath == query2.ApiPath_CDocs {
				return fmt.Sprintf("Reads the collection of %s", typeName)
			}
			return fmt.Sprintf("Reads %s", typeName)
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

func (g *schemaGenerator) schemaNameByTypeName(typeName string, op appdef.OperationKind) string {
	if typeSchemas, ok := g.schemasByType[typeName]; ok {
		if opSchema, ok := typeSchemas[op]; ok {
			return opSchema
		}
	}
	return typeName
}

// generateRequestBody creates a request body for a type and operation
func (g *schemaGenerator) generateRequestBody(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	typeName := typ.QName().String()

	if typ.Kind() == appdef.TypeKind_Command {
		cmd := typ.(appdef.ICommand)
		param := cmd.Param()
		if _, ok := param.(appdef.IODoc); !ok {
			return map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": g.generateSchema(param.(ischema), op, nil),
					},
				},
			}
		}
		typeName = param.QName().String()
	}

	return map[string]interface{}{
		"required": true,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": "#/components/schemas/" + g.schemaNameByTypeName(typeName, op),
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
					"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
				},
			},
		},
	}

	responses["403"] = map[string]interface{}{
		"description": "Forbidden",
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
				},
			},
		},
	}

	responses["404"] = map[string]interface{}{
		"description": "Not Found",
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
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
						"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
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
						"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
					},
				},
			},
		}

	case op == appdef.OperationKind_Select && appdef.TypeKind_Records.Contains(typ.Kind()):
		responses["200"] = map[string]interface{}{
			"description": "OK",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/" + g.schemaNameByTypeName(typeName, op),
					},
				},
			},
		}

	case (op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_CDoc) ||
		(op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_ViewRecord):
		// Collection response with results array

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
									"$ref": "#/components/schemas/" + g.schemaNameByTypeName(typeName, op),
								},
							},
							"error": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
							},
						},
					},
				},
			},
		}

	case (op == appdef.OperationKind_Execute && typ.Kind() == appdef.TypeKind_Query):
		// Collection response with results array
		schemaRef := g.schemaNameByTypeName(typ.(appdef.IQuery).Result().QName().String(), op)

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
									"$ref": "#/components/schemas/" + schemaRef,
								},
							},
							"error": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s", errorSchemaName),
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
