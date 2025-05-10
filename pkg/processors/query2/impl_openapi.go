/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/processors"
)

// [~server.apiv2.role/cmp.CreateOpenApiSchema~impl]
// CreateOpenAPISchema generates an OpenAPI schema document for the given workspace and role
func CreateOpenAPISchema(writer io.Writer, ws appdef.IWorkspace, role appdef.QName,
	pubTypesFunc PublishedTypesFunc, meta SchemaMeta, developer bool) error {

	generator := &schemaGenerator{
		developer:      developer,
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
	developer    bool
	ws           appdef.IWorkspace
	role         appdef.QName
	pubTypesFunc PublishedTypesFunc
	meta         SchemaMeta
	types        iter.Seq2[appdef.IType,
		iter.Seq2[appdef.OperationKind, *[]appdef.FieldName]]
	components     map[string]interface{}
	paths          map[string]map[string]interface{}
	schemasByType  map[string]map[appdef.OperationKind]string
	docTypes       map[appdef.QName]map[appdef.OperationKind]bool // bool: true when operation is allowed on limited fields
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
	g.components["securitySchemes"] = map[string]interface{}{
		bearerAuth: map[string]interface{}{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		},
	}
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
				cmd := t.(appdef.ICommand)
				if param := cmd.Param(); param != nil {
					g.generateSchemaComponent(param.(ischema), op, nil, schemas)
				}
				if result := cmd.Result(); result != nil {
					g.generateSchemaComponent(result.(ischema), op, nil, schemas)
				}
			}
			if t.Kind() == appdef.TypeKind_Query && op == appdef.OperationKind_Execute {
				qry := t.(appdef.IQuery)
				if param := qry.Param(); param != nil {
					g.generateSchemaComponent(param.(ischema), op, nil, schemas)
				}
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

	schemas[registeredDeviceSchemaName] = map[string]interface{}{
		schemaKeyType: schemaTypeObject,
		schemaKeyProperties: map[string]interface{}{
			fieldLogin: map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
			fieldPassword: map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
			"ProfileWSID": map[string]interface{}{
				schemaKeyType:   schemaTypeInteger,
				schemaKeyFormat: schemaFormatInt64,
			},
		},
	}

	// [~server.apiv2.auth/cmp.principalTokenSchema~impl]
	schemas[principalTokenSchemaName] = map[string]interface{}{
		schemaKeyType: schemaTypeObject,
		schemaKeyProperties: map[string]interface{}{
			"PrincipalToken": map[string]interface{}{
				schemaKeyType: schemaTypeString,
			},
			"ExpiresIn": map[string]interface{}{
				schemaKeyType:   schemaTypeInteger,
				schemaKeyFormat: schemaFormatInt64,
			},
			"WSID": map[string]interface{}{
				schemaKeyType:   schemaTypeInteger,
				schemaKeyFormat: schemaFormatInt64,
			},
		},
	}

}

func (g *schemaGenerator) collectDocSchemaTypes() {
	g.docTypes = make(map[appdef.QName]map[appdef.OperationKind]bool)
	for t, ops := range g.types {
		if appdef.TypeKind_Docs.Contains(t.Kind()) || appdef.TypeKind_Records.Contains(t.Kind()) {
			opsa := make(map[appdef.OperationKind]bool)
			for op, fields := range ops {
				opsa[op] = fields != nil && len(*fields) > 0
			}
			g.docTypes[t.QName()] = opsa
		}
	}
}

func (g *schemaGenerator) opString(op appdef.OperationKind) string {
	switch op {
	case appdef.OperationKind_Insert:
		return "Insert"
	case appdef.OperationKind_Update:
		return "Update"
	case appdef.OperationKind_Deactivate:
		return "Deactivate"
	case appdef.OperationKind_Select:
		return "Select"
	case appdef.OperationKind_Execute:
		return "Execute"
	default:
		return "Unknown"
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
		componentName = fmt.Sprintf("%s_%s", typeName, g.opString(op))
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
	g.addAuthPaths()
	if g.developer {
		g.genCreateNewUserPath()
		g.genCreateNewDevicePath()
	}
	for t, ops := range g.types {
		for op := range ops {
			paths := g.getPaths(t, op)
			for _, path := range paths {
				g.addPathItem(path.Path, path.Method, t, op, path.APIPath)
			}
		}
	}
}

func (g *schemaGenerator) addAuthPaths() {
	g.genAuthLoginPath()
	g.genAuthRefreshPath()
}

func (g *schemaGenerator) genCreateNewUserPath() {
	path := fmt.Sprintf("/api/v2/apps/%s/%s/users", g.getOwner(), g.getApp())
	g.paths[path] = map[string]interface{}{
		schemaMethodPost: map[string]interface{}{
			schemaKeyDescription: "Register a new user with the provided details",
			schemaKeyTags:        []string{usersTag},
			schemaKeyParameters:  g.generateParameters(path, nil),
			schemaKeyRequestBody: map[string]interface{}{
				schemaKeyRequired: true,
				schemaKeyContent: map[string]interface{}{
					applicationJSON: map[string]interface{}{
						schemaKeySchema: map[string]interface{}{
							schemaKeyType: schemaTypeObject,
							schemaKeyProperties: map[string]interface{}{
								fieldVerifiedEmailToken: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
								fieldPassword: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
								fieldDisplayName: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
							},
							schemaKeyRequired: []string{fieldVerifiedEmailToken, fieldPassword, fieldDisplayName},
						},
					},
				},
			},
			schemaKeyResponses: map[string]interface{}{
				statusCode201: g.genOKResponse(""),
				statusCode400: g.genErrorResponse(http.StatusBadRequest),
				statusCode401: g.genErrorResponse(http.StatusUnauthorized),
				statusCode429: g.genErrorResponse(http.StatusTooManyRequests),
			},
		},
	}
}

func (g *schemaGenerator) genCreateNewDevicePath() {
	path := fmt.Sprintf("/api/v2/apps/%s/%s/devices", g.getOwner(), g.getApp())
	g.paths[path] = map[string]interface{}{
		schemaMethodPost: map[string]interface{}{
			schemaKeyDescription: "Create(register) new device",
			schemaKeySecurity: []map[string]interface{}{
				{
					bearerAuth: []string{},
				},
			},
			schemaKeyTags:       []string{devicesTag},
			schemaKeyParameters: g.generateParameters(path, nil),
			schemaKeyRequestBody: map[string]interface{}{
				schemaKeyRequired: true,
				schemaKeyContent: map[string]interface{}{
					applicationJSON: map[string]interface{}{
						schemaKeySchema: map[string]interface{}{
							schemaKeyType: schemaTypeObject,
							schemaKeyProperties: map[string]interface{}{
								fieldDisplayName: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
							},
							schemaKeyRequired: []string{fieldDisplayName},
						},
					},
				},
			},
			schemaKeyResponses: map[string]interface{}{
				statusCode201: g.genOKResponse(registeredDeviceSchemaRef),
				statusCode400: g.genErrorResponse(http.StatusBadRequest),
				statusCode401: g.genErrorResponse(http.StatusUnauthorized),
				statusCode403: g.genErrorResponse(http.StatusForbidden),
				statusCode429: g.genErrorResponse(http.StatusTooManyRequests),
			},
		},
	}
}

func (g *schemaGenerator) genAuthLoginPath() {
	// [~server.apiv2.auth/cmp.provideAuthLoginPath~impl]
	path := fmt.Sprintf("/api/v2/apps/%s/%s/auth/login", g.getOwner(), g.getApp())
	parameters := g.generateParameters(path, nil)
	g.paths[path] = map[string]interface{}{
		schemaMethodPost: map[string]interface{}{
			schemaKeyDescription: "Issues (creates) a new principal token in exchange for valid credentials",
			schemaKeyTags:        []string{authenticationTag},
			schemaKeyParameters:  parameters,
			schemaKeyRequestBody: map[string]interface{}{
				schemaKeyRequired: true,
				schemaKeyContent: map[string]interface{}{
					applicationJSON: map[string]interface{}{
						schemaKeySchema: map[string]interface{}{
							schemaKeyType: schemaTypeObject,
							schemaKeyProperties: map[string]interface{}{
								// Login is a mandatory field
								fieldLogin: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
								fieldPassword: map[string]interface{}{
									schemaKeyType: schemaTypeString,
								},
							},
							schemaKeyRequired: []string{fieldLogin, fieldPassword},
						},
					},
				},
			},
			schemaKeyResponses: map[string]interface{}{
				statusCode200: g.genOKResponse(principalTokenSchemaRef),
				statusCode400: g.genErrorResponse(http.StatusBadRequest),
				statusCode401: g.genErrorResponse(http.StatusUnauthorized),
				statusCode429: g.genErrorResponse(http.StatusTooManyRequests),
			},
		},
	}
}

func (g *schemaGenerator) genAuthRefreshPath() {
	// [~server.apiv2.auth/cmp.provideAuthRefreshPath~impl]
	path := fmt.Sprintf("/api/v2/apps/%s/%s/auth/refresh", g.getOwner(), g.getApp())
	parameters := g.generateParameters(path, nil)
	g.paths[path] = map[string]interface{}{
		schemaMethodPost: map[string]interface{}{
			schemaKeyDescription: "Returns a refreshed principal token",
			schemaKeySecurity: []map[string]interface{}{
				{
					bearerAuth: []string{},
				},
			},
			schemaKeyTags:       []string{authenticationTag},
			schemaKeyParameters: parameters,
			schemaKeyResponses: map[string]interface{}{
				statusCode200: g.genOKResponse(principalTokenSchemaRef),
				statusCode400: g.genErrorResponse(http.StatusBadRequest),
				statusCode401: g.genErrorResponse(http.StatusUnauthorized),
				statusCode403: g.genErrorResponse(http.StatusForbidden),
				statusCode429: g.genErrorResponse(http.StatusTooManyRequests),
			},
		},
	}
}

func (g *schemaGenerator) genOKResponse(schemaRef string) map[string]interface{} {
	if schemaRef == "" {
		return map[string]interface{}{
			schemaKeyDescription: descrOK,
		}
	}

	return map[string]interface{}{
		schemaKeyDescription: descrOK,
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: schemaRef,
				},
			},
		},
	}
}

func (g *schemaGenerator) genErrorResponse(code int) map[string]interface{} {
	var description string
	switch code {
	case http.StatusBadRequest:
		description = "Bad request"
	case http.StatusUnauthorized:
		description = "Unauthorized"
	case http.StatusForbidden:
		description = "Forbidden"
	case http.StatusNotFound:
		description = "Not found"
	case http.StatusInternalServerError:
		description = "Internal server error"
	case http.StatusTooManyRequests:
		description = "Too many requests"
	default:
		description = "Unknown error"
	}
	return map[string]interface{}{
		schemaKeyDescription: description,
		schemaKeyContent: map[string]interface{}{
			applicationJSON: map[string]interface{}{
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: errorSchemaRef,
				},
			},
		},
	}
}

func (g *schemaGenerator) getOwner() string {
	if g.meta.AppName != appdef.NullAppQName {
		return g.meta.AppName.Owner()
	}
	return "{owner}"
}

func (g *schemaGenerator) getApp() string {
	if g.meta.AppName != appdef.NullAppQName {
		return g.meta.AppName.Name()
	}
	return "{app}"
}

// getPathAndMethod returns the API path and HTTP method for a given type and operation
func (g *schemaGenerator) getPaths(typ appdef.IType, op appdef.OperationKind) []pathItem {
	typeName := typ.QName().String()

	switch typ.Kind() {
	case appdef.TypeKind_CDoc, appdef.TypeKind_WDoc, appdef.TypeKind_CRecord, appdef.TypeKind_WRecord:
		switch op {
		case appdef.OperationKind_Insert:
			return []pathItem{
				{
					Method:  methodPost,
					Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/docs/%s", g.getOwner(), g.getApp(), typeName),
					APIPath: processors.APIPath_Docs,
				},
			}
		case appdef.OperationKind_Update:
			return []pathItem{
				{
					Method:  methodPatch,
					Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/docs/%s/{id}", g.getOwner(), g.getApp(), typeName),
					APIPath: processors.APIPath_Docs,
				},
			}
		case appdef.OperationKind_Deactivate:
			return []pathItem{
				{
					Method:  methodDelete,
					Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/docs/%s/{id}", g.getOwner(), g.getApp(), typeName),
					APIPath: processors.APIPath_Docs,
				},
			}
		case appdef.OperationKind_Select:
			return []pathItem{
				{
					Method:  methodGet,
					Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/docs/%s/{id}", g.getOwner(), g.getApp(), typeName),
					APIPath: processors.APIPath_Docs,
				},
				{
					Method:  methodGet,
					Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/cdocs/%s", g.getOwner(), g.getApp(), typeName),
					APIPath: processors.APIPath_CDocs,
				},
			}

		}
	}

	if _, ok := typ.(appdef.ICommand); ok && op == appdef.OperationKind_Execute {
		return []pathItem{
			{
				Method:  methodPost,
				Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/commands/%s", g.getOwner(), g.getApp(), typeName),
				APIPath: processors.APIPath_Commands,
			},
		}
	}

	if _, ok := typ.(appdef.IQuery); ok && op == appdef.OperationKind_Execute {
		return []pathItem{
			{
				Method:  methodGet,
				Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/queries/%s", g.getOwner(), g.getApp(), typeName),
				APIPath: processors.APIPath_Queries,
			},
		}
	}

	if _, ok := typ.(appdef.IView); ok && op == appdef.OperationKind_Select {
		return []pathItem{
			{
				Method:  methodGet,
				Path:    fmt.Sprintf("/apps/%s/%s/workspaces/{wsid}/views/%s", g.getOwner(), g.getApp(), typeName),
				APIPath: processors.APIPath_Views,
			},
		}
	}

	return nil
}

// addPathItem adds a path item to the OpenAPI schema
func (g *schemaGenerator) addPathItem(path, method string, typ appdef.IType, op appdef.OperationKind, apiPath processors.APIPath) {

	// Create path if it doesn't exist
	if _, exists := g.paths[path]; !exists {
		g.paths[path] = make(map[string]interface{})
	}

	// Create operation object
	operation := make(map[string]interface{})

	// Add tags based on type's tags
	tags := g.generateTags(typ)
	if len(tags) > 0 {
		operation[schemaKeyTags] = tags
	}

	// Add operation description
	operation[schemaKeyDescription] = g.generateDescription(typ, op, apiPath)
	operation[schemaKeySecurity] = []map[string]interface{}{
		{
			bearerAuth: []string{},
		},
	}

	// Add operation parameters
	parameters := g.generateParameters(path, typ)
	if len(parameters) > 0 {
		operation[schemaKeyParameters] = parameters
	}

	// Add request body for appropriate methods
	if method == "post" || method == "patch" || method == "put" {
		requestBody := g.generateRequestBody(typ, op)
		if requestBody != nil {
			operation[schemaKeyRequestBody] = requestBody
		}
	}

	// Add responses
	operation[schemaKeyResponses] = g.generateResponses(typ, op, apiPath)

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
func (g *schemaGenerator) generateDescription(typ appdef.IType, op appdef.OperationKind, apiPath processors.APIPath) string {
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
			if apiPath == processors.APIPath_CDocs {
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
func (g *schemaGenerator) generateParameters(path string, typ appdef.IType) []map[string]interface{} {
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
			schemaKeyDescription: "Name of a user who owns the application",
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
			schemaKeyDescription: "Name of an application",
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
			schemaKeyDescription: "The ID of workspace",
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
			schemaKeyDescription: "ID of a document or record",
		})
	}

	// Add query parameters for GET methods
	if strings.Contains(path, "/views/") || strings.Contains(path, "/queries/") || strings.Contains(path, "/cdocs/") {
		// Add query constraints parameters
		pkFields := make([]string, 0)
		if view, ok := typ.(appdef.IView); ok {
			for _, pk := range view.Key().PartKey().Fields() {
				pkFields = append(pkFields, pk.Name())
			}
		}
		descr := "A JSON-encoded string used to filter query results. The value must be URL-encoded"
		if len(pkFields) > 0 {
			descr += fmt.Sprintf(". Required fields: %s", strings.Join(pkFields, ", "))
		}
		parameters = append(parameters, map[string]interface{}{
			"name":     "where",
			"in":       "query",
			"required": len(pkFields) > 0,
			schemaKeySchema: map[string]interface{}{
				"type": "string",
			},
			schemaKeyDescription: descr,
			"example":            `{"Country": "Spain", "Age": {"$gt": 30}}`,
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "order",
			"in":       "query",
			"required": false,
			schemaKeySchema: map[string]interface{}{
				"type": "string",
			},
			schemaKeyDescription: "Field to order results by",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "limit",
			"in":       "query",
			"required": false,
			schemaKeySchema: map[string]interface{}{
				"type": "integer",
			},
			schemaKeyDescription: "Maximum number of results to return",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "skip",
			"in":       "query",
			"required": false,
			schemaKeySchema: map[string]interface{}{
				"type": "integer",
			},
			schemaKeyDescription: "Number of results to skip",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "include",
			"in":       "query",
			"required": false,
			schemaKeySchema: map[string]interface{}{
				"type": "string",
			},
			schemaKeyDescription: "Referenced objects to include in response",
		})

		parameters = append(parameters, map[string]interface{}{
			"name":     "keys",
			"in":       "query",
			"required": false,
			schemaKeySchema: map[string]interface{}{
				"type": "string",
			},
			schemaKeyDescription: "Specific fields to include in response",
		})
	}

	// Add arg parameter for queries
	if strings.Contains(path, "/queries/") {
		query := typ.(appdef.IQuery)
		if query.Param() != nil {
			parameters = append(parameters, map[string]interface{}{
				"name":     "args",
				"in":       "query",
				"required": true,
				schemaKeySchema: map[string]interface{}{
					schemaKeyRef: g.schemaRef(query.Param(), appdef.OperationKind_Execute),
				},
				schemaKeyDescription: "Query argument in JSON format",
			})
		}
	}

	return parameters
}

func (g *schemaGenerator) docSchemaRefIfExist(typ appdef.QName, op appdef.OperationKind) (string, bool) {
	ops, ok := g.docTypes[typ]
	if !ok {
		return "", false
	}
	if limited, ok := ops[op]; ok {
		if limited {
			return fmt.Sprintf("#/components/schemas/%s_%s", typ.String(), g.opString(op)), true
		}
		return fmt.Sprintf("#/components/schemas/%s", typ.String()), true
	}
	return "", false
}

func (g *schemaGenerator) schemaRef(typ appdef.IType, op appdef.OperationKind) string {
	if typeSchemas, ok := g.schemasByType[typ.QName().String()]; ok {
		if opSchema, ok := typeSchemas[op]; ok {
			return fmt.Sprintf("#/components/schemas/%s", opSchema)
		}
	}
	return fmt.Sprintf("#/components/schemas/%s", typ.QName().String())
}

// generateRequestBody creates a request body for a type and operation
func (g *schemaGenerator) generateRequestBody(typ appdef.IType, op appdef.OperationKind) map[string]interface{} {
	if typ.Kind() == appdef.TypeKind_Command {
		cmd := typ.(appdef.ICommand)
		param := cmd.Param()
		unloggedParam := cmd.UnloggedParam()
		properties := make(map[string]interface{})

		if _, ok := param.(appdef.IODoc); !ok && param != nil {
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
func (g *schemaGenerator) generateResponses(typ appdef.IType, op appdef.OperationKind, apiPath processors.APIPath) map[string]interface{} {
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

	case apiPath == processors.APIPath_Docs &&
		op == appdef.OperationKind_Select &&
		appdef.TypeKind_Records.Contains(typ.Kind()):
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

	case apiPath == processors.APIPath_CDocs && op == appdef.OperationKind_Select &&
		(typ.Kind() == appdef.TypeKind_CDoc || typ.Kind() == appdef.TypeKind_ViewRecord):
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
			"contact": map[string]interface{}{
				"name": g.meta.AppName.Owner(),
			},
			schemaKeyDescription: g.meta.Description,
		},
		"externalDocs": map[string]interface{}{
			schemaKeyDescription: "Powered by Voedger: distributed cloud application platform",
			"url":                "https://voedger.io",
		},
		"servers": []map[string]interface{}{
			{
				"url": "/api/v2",
			},
		},
		"paths":      g.paths,
		"components": g.components,
	}

	// Serialize to JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schema)
}
