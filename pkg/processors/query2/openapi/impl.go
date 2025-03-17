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

	generator.generate()

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
func (g *schemaGenerator) generate() {
	g.types = g.pubTypesFunc(g.ws, g.role)
	g.collectDocSchemaTypes()

	// First pass - generate schema components for types
	g.generateComponents()

	// Second pass - generate paths using components
	g.generatePaths()
}

// generateComponents creates schema components for all published types
func (g *schemaGenerator) generateComponents() {
	schemas := make(map[string]interface{})
	g.components["schemas"] = schemas

	for t, ops := range g.types {
		typeName := t.QName().String()

		// Skip if already processed
		if g.processedTypes[typeName] {
			continue
		}

		for op, fields := range ops {
			g.generateSchemaComponent(t, op, fields, schemas)
			if t.Kind() == appdef.TypeKind_Command && op == appdef.OperationKind_Execute {
				// If command param is an ODoc, generate a schema for it
				cmd := t.(appdef.ICommand)
				param := cmd.Param()
				if _, ok := param.(appdef.IODoc); ok {
					g.generateSchemaComponent(param.(ischema), op, nil, schemas)
				}
				if result := cmd.Result(); result != nil {
					g.generateSchemaComponent(result.(ischema), op, nil, schemas)
				}
			}
			if t.Kind() == appdef.TypeKind_Query && op == appdef.OperationKind_Execute {
				qry := t.(appdef.IQuery)
				if result := qry.Result(); result != nil {
					g.generateSchemaComponent(result.(ischema), op, nil, schemas)
				}
			}
		}

		g.processedTypes[typeName] = true
	}

	// generate error schema
	schemas[errorSchemaName] = map[string]interface{}{
		schemaKeyType: schemaTypeObject,
		schemaKeyProperties: map[string]interface{}{
			"message": map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
			"status": map[string]interface{}{
				schemaKeyType: schemaTypeInteger,
			},
			"qname": map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
			"data": map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
		},
		schemaKeyRequired: []string{"message"},
	}
}

func (g *schemaGenerator) collectDocSchemaTypes() {
	g.docTypes = make(map[appdef.QName]bool)
	for t := range g.types {
		if appdef.TypeKind_Docs.Contains(t.Kind()) || appdef.TypeKind_Records.Contains(t.Kind()) {
			g.docTypes[t.QName()] = true
		}
	}
}

// generateSchemaComponent creates a schema component for a specific type and operation
func (g *schemaGenerator) generateSchemaComponent(typ appdef.IType, op appdef.OperationKind,
	fieldNames *[]appdef.FieldName, schemas map[string]interface{}) {

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
		return // Type doesn't have fields, skip
	}

	schemas[componentName] = g.generateSchema(withFields, op, fieldNames)

	if typ.Kind() == appdef.TypeKind_ODoc || typ.Kind() == appdef.TypeKind_ORecord {
		// Generate schema components ODoc inner containers
		withContainers := typ.(appdef.IWithContainers)
		for _, container := range withContainers.Containers() {
			g.generateSchemaComponent(container.Type(), op, nil, schemas)
		}
	}
}

// generatePaths creates path items for all published types and their operations
func (g *schemaGenerator) generatePaths() {
	for t, ops := range g.types {
		for op := range ops {
			paths := g.getPaths(t, op)
			for _, path := range paths {
				g.addPathItem(path.Path, path.Method, t, op, path.ApiPath)
			}
		}
	}
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
func (g *schemaGenerator) addPathItem(path, method string, typ appdef.IType, op appdef.OperationKind, apiPath query2.ApiPath) {
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
			}
			return fmt.Sprintf("Selects from query %s", typeName)
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

func (g *schemaGenerator) schemaRef(typ appdef.IType, op appdef.OperationKind) string {
	return fmt.Sprintf("#/components/schemas/%s", g.schemaNameByTypeName(typ.QName().String(), op))
}

// generateRequestBody creates a request body for a type and operation
func (g *schemaGenerator) generateRequestBody(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	if typ.Kind() == appdef.TypeKind_Command {
		cmd := typ.(appdef.ICommand)
		param := cmd.Param()
		unloggedParam := cmd.UnloggedParam()
		properties := make(map[string]interface{})

		if _, ok := param.(appdef.IODoc); !ok {
			properties["args"] = map[string]interface{}{
				schemaKeyRef: g.schemaRef(param, op),
			}
		} else if param != nil {
			properties["args"] = g.generateSchema(param.(ischema), op, nil)
		}

		if unloggedParam != nil {
			properties["unloggedArgs"] = g.generateSchema(unloggedParam.(ischema), op, nil)
		}

		return map[string]interface{}{
			"required": true,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType:       schemaTypeObject,
						schemaKeyProperties: properties,
					},
				},
			},
		}
	}

	return map[string]interface{}{
		"required": true,
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: g.schemaRef(typ, op),
				},
			},
		},
	}
}

// generateResponses creates response objects for a type and operation
func (g *schemaGenerator) generateResponses(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	responses := make(map[string]interface{})
	// Add standard error responses
	responses["401"] = map[string]interface{}{
		schemaKeyDescription: "Unauthorized",
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: errorSchemaRef,
				},
			},
		},
	}

	responses["403"] = map[string]interface{}{
		schemaKeyDescription: "Forbidden",
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: errorSchemaRef,
				},
			},
		},
	}

	responses["404"] = map[string]interface{}{
		schemaKeyDescription: "Not Found",
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: errorSchemaRef,
				},
			},
		},
	}

	// Add specific successful response based on type and operation
	switch {
	case op == appdef.OperationKind_Insert:
		responses["201"] = map[string]interface{}{
			schemaKeyDescription: "Created",
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType: schemaTypeObject,
						schemaKeyProperties: map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								schemaKeyType: schemaTypeInteger,
							},
							"NewIDs": map[string]interface{}{
								schemaKeyType: schemaTypeObject,
								"additionalProperties": map[string]interface{}{
									schemaKeyType:   schemaTypeInteger,
									schemaKeyFormat: schemaFormatInt64,
								},
							},
						},
					},
				},
			},
		}

		// Add bad request response for create operations
		responses["400"] = map[string]interface{}{
			schemaKeyDescription: "Bad Request",
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyRef: errorSchemaRef,
					},
				},
			},
		}

	case op == appdef.OperationKind_Update || op == appdef.OperationKind_Deactivate:
		responses[statusCode200] = map[string]interface{}{
			schemaKeyDescription: descrOK,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType: schemaTypeObject,
						schemaKeyProperties: map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								schemaKeyType: schemaTypeInteger,
							},
						},
					},
				},
			},
		}

	case op == appdef.OperationKind_Execute && typ.Kind() == appdef.TypeKind_Command:
		responses[statusCode200] = map[string]interface{}{
			schemaKeyDescription: descrOK,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType: schemaTypeObject,
						schemaKeyProperties: map[string]interface{}{
							"CurrentWLogOffset": map[string]interface{}{
								schemaKeyType: schemaTypeInteger,
							},
						},
					},
				},
			},
		}

		// Add bad request response for command execution
		responses["400"] = map[string]interface{}{
			schemaKeyDescription: "Bad Request",
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyRef: errorSchemaRef,
					},
				},
			},
		}

	case op == appdef.OperationKind_Select && appdef.TypeKind_Records.Contains(typ.Kind()):
		responses[statusCode200] = map[string]interface{}{
			schemaKeyDescription: descrOK,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyRef: g.schemaRef(typ, op),
					},
				},
			},
		}

	case (op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_CDoc) ||
		(op == appdef.OperationKind_Select && typ.Kind() == appdef.TypeKind_ViewRecord):
		// Collection response with results array

		responses[statusCode200] = map[string]interface{}{
			schemaKeyDescription: descrOK,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType: schemaTypeObject,
						schemaKeyProperties: map[string]interface{}{
							"results": map[string]interface{}{
								schemaKeyType: schemaTypeArray,
								schemaKeyItems: map[string]interface{}{
									schemaKeyRef: g.schemaRef(typ, op),
								},
							},
							"error": map[string]interface{}{
								schemaKeyRef: errorSchemaRef,
							},
						},
					},
				},
			},
		}

	case (op == appdef.OperationKind_Execute && typ.Kind() == appdef.TypeKind_Query):
		// Collection response with results array
		responses[statusCode200] = map[string]interface{}{
			schemaKeyDescription: descrOK,
			schemaKeyContent: map[string]interface{}{
				applicationJSON: map[string]interface{}{
					schemaKeySchema: map[string]interface{}{
						schemaKeyType: schemaTypeObject,
						schemaKeyProperties: map[string]interface{}{
							"results": map[string]interface{}{
								schemaKeyType: schemaTypeArray,
								schemaKeyItems: map[string]interface{}{
									schemaKeyRef: g.schemaRef(typ.(appdef.IQuery).Result(), op),
								},
							},
							"error": map[string]interface{}{
								schemaKeyRef: errorSchemaRef,
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
