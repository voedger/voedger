# OpenAPI Schema Generator

This package provides functionality to generate OpenAPI 3.0 schemas from appdef types, allowing you to expose your application's API using standard OpenAPI documentation.

## Overview

The OpenAPI generator automatically creates API documentation for:
- Documents (CDoc, WDoc) and Records (CRecord, WRecord)
- Commands
- Queries
- Views

It generates appropriate paths, operations, parameters, request bodies, and responses based on the appdef types and their operations.

## Usage

### Main Function

```go
func CreateOpenApiSchema(
    writer io.Writer, 
    ws appdef.IWorkspace, 
    role appdef.QName,
    pubTypesFunc PublishedTypesFunc, 
    meta SchemaMeta,
) error
```

This function generates an OpenAPI schema document for a given workspace and role.

### Parameters

- `writer`: The output writer where the OpenAPI schema will be written (typically a file or HTTP response)
- `ws`: The workspace containing the types to document
- `role`: The role for which to generate the schema (determines which types are accessible)
- `pubTypesFunc`: A function that returns publishable types with their operations and field constraints
- `meta`: Metadata for the schema (title, version, application name)

### Example

```go
package main

import (
    "os"
    "iter"
    
    "github.com/voedger/voedger/pkg/appdef"
    "github.com/voedger/voedger/pkg/appdef/acl"
    "github.com/voedger/voedger/pkg/processors/query2/openapi"
)

func main() {
    // Open file for writing
    file, err := os.Create("openapi.json")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    
    // Define metadata
    meta := openapi.SchemaMeta{
        SchemaTitle:   "My API Documentation",
        SchemaVersion: "1.0.0",
        AppName:       appdef.NewAppQName("mycompany", "myapp"),
    }
    
    // Create schema
    err = openapi.CreateOpenApiSchema(
        file,
        myWorkspace,
        appdef.QName_Role_User,
        acl.PublishedTypes,
        meta,
    )
    
    if err != nil {
        panic(err)
    }
}

## SchemaMeta Details

```go
type SchemaMeta struct {
    SchemaTitle   string              // Title of the OpenAPI document
    SchemaVersion string              // Version of the API
    AppName       appdef.AppQName     // Application name (owner.app)
}
```

