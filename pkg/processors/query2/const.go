/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	_ "embed"

	"github.com/voedger/voedger/pkg/appdef"
)

//go:embed swagger-ui.html
var swaggerUI_HTML string

const (
	errorSchemaName            = "Error"
	errorSchemaRef             = "#/components/schemas/" + errorSchemaName
	principalTokenSchemaName   = "PrincipalToken"
	principalTokenSchemaRef    = "#/components/schemas/" + principalTokenSchemaName
	registeredDeviceSchemaName = "CreateDeviceResult"
	registeredDeviceSchemaRef  = "#/components/schemas/" + registeredDeviceSchemaName
	bearerAuth                 = "BearerAuth"
	cookieAuth                 = "CookieAuth"
	authenticationTag          = "Authentication"
	usersTag                   = "Users"
	devicesTag                 = "Devices"
	BlobCreateResultSchemaName = "BlobCreateResult"
	BlobCreateResultSchemaRef  = "#/components/schemas/" + BlobCreateResultSchemaName
)

var qNameWDocBLOB = appdef.NewQName(appdef.SysPackage, "BLOB")
var sysAnySchemaName = appdef.QNameANY.String()
var sysAnySchemaRef = "#/components/schemas/" + sysAnySchemaName

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
	statusCode201 = "201"
	statusCode400 = "400"
	statusCode401 = "401"
	statusCode403 = "403"
	statusCode404 = "404"
	statusCode413 = "413"
	statusCode415 = "415"
	statusCode429 = "429"
	statusCode500 = "500"
	statusCode503 = "503"
)

// OpenAPI schema constants
const (
	schemaTypeObject  = "object"
	schemaTypeString  = "string"
	schemaTypeInteger = "integer"
	schemaTypeNumber  = "number"
	schemaTypeBoolean = "boolean"
	schemaTypeArray   = "array"

	schemaMethodPost = "post"
	schemaMethodGet  = "get"

	schemaFormatInt32  = "int32"
	schemaFormatInt64  = "int64"
	schemaFormatFloat  = "float"
	schemaFormatDouble = "double"
	schemaFormatByte   = "byte"
	schemaFormatBinary = "binary"

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
	schemaKeyRequestBody = "requestBody"
	schemaKeyResponses   = "responses"
	schemaKeyParameters  = "parameters"
	schemaKeySecurity    = "security"
	schemaKeyTags        = "tags"
)

// HTTP Headers
const (
	headerContentType = "Content-Type"
	headerBlobName    = "Blob-Name"
)

// Content type values
const (
	applicationOctetStream = "application/octet-stream"
)

// Common field names and properties
const (
	fieldLogin              = "login"
	fieldPassword           = "password"
	fieldDisplayName        = "displayName"
	fieldAppName            = "appName"
	fieldVerifiedEmailToken = "verifiedEmailToken"
	fieldMessage            = "message"
	fieldStatus             = "status"
	fieldQName              = "qname"
	fieldData               = "data"
	fieldBlobID             = "blobID"
	fieldPrincipalToken     = "principalToken"
	fieldExpiresInSeconds   = "expiresInSeconds"
	fieldProfileWSID        = "profileWSID"
	fieldCurrentWLogOffset  = "currentWLogOffset"
	fieldNewIDs             = "newIDs"
	fieldResults            = "results"
	fieldError              = "error"
	fieldArgs               = "args"
	fieldUnloggedArgs       = "unloggedArgs"
)

// Parameter names
const (
	paramOwner   = "owner"
	paramApp     = "app"
	paramWSID    = "wsid"
	paramID      = "id"
	paramWhere   = "where"
	paramOrder   = "order"
	paramLimit   = "limit"
	paramSkip    = "skip"
	paramInclude = "include"
	paramKeys    = "keys"
	paramArgs    = "args"
)

// Parameter locations
const (
	paramInPath   = "path"
	paramInQuery  = "query"
	paramInHeader = "header"
)

// OpenAPI schema properties
const (
	propertyMinItems             = "minItems"
	propertyMaxItems             = "maxItems"
	propertyMaxLength            = "maxLength"
	propertyPattern              = "pattern"
	propertyExample              = "example"
	propertyHeaders              = "headers"
	propertyAdditionalProperties = "additionalProperties"
)

// Default values and examples
const (
	qNamePatternRegex = "^[a-zA-Z0-9_]+\\.[a-zA-Z0-9_]+$"
	qNameExample      = "app1pkg.MyType"
	whereExample      = `{"Country": "Spain", "Age": {"$gt": 30}}`
)

// OpenAPI specification constants
const (
	openAPIVersion = "3.0.0"
	bearerScheme   = "bearer"
	httpType       = "http"
	jwtFormat      = "JWT"
)

// Error descriptions
const (
	descrBadRequest           = "Bad request"
	descrUnauthorized         = "Unauthorized"
	descrForbidden            = "Forbidden"
	descrNotFound             = "Not found"
	descrInternalServerError  = "Internal server error"
	descrTooManyRequests      = "Too many requests"
	descrPayloadTooLarge      = "Payload Too Large"
	descrUnsupportedMediaType = "Unsupported Media Type"
	descrServiceUnavailable   = "Service Unavailable"
	descrUnknownError         = "Unknown error"
)

// API path templates
const (
	pathTemplateOwner = "{owner}"
	pathTemplateApp   = "{app}"
	pathTemplateWSID  = "{wsid}"
	pathTemplateID    = "{id}"
)

// Description templates
const (
	descrOwnerParam = "Name of a user who owns the application"
	descrAppParam   = "Name of an application"
	descrWSIDParam  = "The ID of workspace"

	descrWhereParam       = "A JSON-encoded string used to filter query results. The value must be URL-encoded"
	descrOrderParam       = "Field to order results by"
	descrLimitParam       = "Maximum number of results to return"
	descrSkipParam        = "Number of results to skip"
	descrIncludeParam     = "Referenced objects to include in response"
	descrKeysParam        = "Specific fields to include in response"
	descrArgsParam        = "Query argument in JSON format"
	descrBlobData         = "BLOB binary data"
	descrBlobContentType  = "BLOB content type"
	descrBlobName         = "BLOB name"
	descrBlobNameOptional = "BLOB name, optional"
	descrCreateBlob       = "Creates a new BLOB for field '%s' of %s"
	descrDownloadBlob     = "Downloads BLOB from field '%s' of %s"
	descrRefToDocRecord   = "Reference to a document or record"
	descrIDOf             = "ID of: %s"
)
