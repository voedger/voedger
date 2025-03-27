/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import _ "embed"

//go:embed swagger-ui.html
var swaggerUi string

const (
	contentTypeHtml = "text/html"
)

const (
	errorSchemaName = "Error"
	errorSchemaRef  = "#/components/schemas/" + errorSchemaName
	bearerAuth      = "BearerAuth"
)

const (
	methodGet    = "get"
	methodPost   = "post"
	methodPut    = "put"
	methodDelete = "delete"
	methodPatch  = "patch"
)

// Content types
const (
	applicationJSON = "application/json"
)

// Descriptions
const (
	descrOK = "OK"
)

// Status codes
const (
	statusCode200 = "200"
)

// OpenAPI schema constants
const (
	schemaTypeObject  = "object"
	schemaTypeString  = "string"
	schemaTypeInteger = "integer"
	schemaTypeNumber  = "number"
	schemaTypeBoolean = "boolean"
	schemaTypeArray   = "array"

	schemaFormatInt32  = "int32"
	schemaFormatInt64  = "int64"
	schemaFormatFloat  = "float"
	schemaFormatDouble = "double"
	schemaFormatByte   = "byte"

	schemaKeyType        = "type"
	schemaKeyFormat      = "format"
	schemaKeyDescription = "description"
	schemaKeyProperties  = "properties"
	schemaKeyRequired    = "required"
	schemaKeyContent     = "content"
	schemaKeySchema      = "schema"
	schemaKeyItems       = "items"
	schemaKeyOneOf       = "oneOf"
	schemaKeyRef         = "$ref"
)
