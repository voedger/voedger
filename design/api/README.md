# Motivation
- We need good REST API for Voedger
- Old API must still be available until the new one is fully developed, so we can continue with AIR

# Functional Design

## API URL
API URL must support versioning ([example IBM MQ](https://www.ibm.com/docs/en/ibm-mq/9.1?topic=api-rest-versions), [example Chargebee](https://apidocs.chargebee.com/docs/api/)):

- old API is available at `/api/v1/...` (for the period of AIR migration it will be available both on `/api/` and `/api/v1/`)
- new API is available at `/api/v2/...`
    - "v1" is not allowed as an owner name, at least until API "v1" is ready

TODO: add endpoint for the list of supported versions

## REST API Paths

| Action                                                                   | Method | REST API Path                                                                |
|--------------------------------------------------------------------------|--------|------------------------------------------------------------------------------|
| **Docs and records**
| [Create document or record](#create-document-or-record)                  | POST   | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}`      |
| [Update document or record](#update-document-or-record)                  | PATCH  | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}` |
| [Deactivate document or record](#deactivate-document-or-record)          | DELETE | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}` |
| [Read document or record](#read-document-or-record)                      | GET    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}` |
| [Read from CDoc Collection](#read-from-cdoc-collection)                  | GET    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}`      |
| **Extensions**
| [Execute Command](#execute-command)                                      | POST   | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/commands/{pkg}.{command}`|
| [Read from Query](#read-from-query)                                      | GET    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/queries/{pkg}.{query}`   |
| **Views**
| [Read from View](#read-from-view)                                        | GET    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/views/{pkg}.{view}`      |
| **BLOBs**
| [Create/upload a new BLOB](#createupload-a-new-blob)                     | POST   | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs`                   |
| [Retrieve/download the BLOB](#retrievedownload-the-blob)                 | GET    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`          |
| [Update an existing BLOB's metadata](#update-an-existing-blobs-metadata) | PATCH  | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`          |
| [Replace an existing BLOB](#replace-an-existing-blob)                    | PUT    | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`          |
| [Delete a BLOB](#delete-the-existing-blob)                               | DELETE | `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`          |
| **Schemas**
| [Read app workspaces](#read-app-workspaces)                              | GET    | `/api/v2/users/{owner}/apps/{app}/workspaceschemas`                          | 
| [Get workspace schema](#get-workspace-schema)                            | GET    | `/api/v2/users/{owner}/apps/{app}/workspaceschemas/{pkg}.{workspace}`        |


## Query Processor based on GET
Current design of the QueryProcessor based on POST queries. 
However, according to many resources, using POST for queries in RESTful API is not a good practice:
- [Swagger.io: best practices in API design](https://swagger.io/resources/articles/best-practices-in-api-design/)
- [MS Azure Architectural Center: Define API operations in terms of HTTP methods](https://learn.microsoft.com/en-us/azure/architecture/best-practices/api-design#define-api-operations-in-terms-of-http-methods)
- [StackOverflow: REST API using POST instead of GET](https://stackoverflow.com/questions/19637459/rest-api-using-post-instead-of-get)

Also, using GET and POST allows to distinguish between Query and Command processors clearly:

| HTTP Method         | Processor         |
|---------------------|-------------------|
| GET                 | Query Processor   |
| POST, PATCH, DELETE | Command Processor |

> Note: according to RESTful API design, queries should not change the state of the system. Current QueryFunction design allows it to execute commands through HTTP bus.

Another thing is that according to REST best practices, it is not recommended to use verbs in the URL, the resource names should be based on nouns:

[Example Microsoft](https://learn.microsoft.com/en-us/azure/architecture/best-practices/api-design#organize-the-api-design-around-resources):
```
POST https://adventure-works.com/orders // Good
POST https://adventure-works.com/create-order // Avoid
```

Summary, the following Queries in airs-bp3:
```
POST .../IssueLinkDeviceToken
POST .../GetSalesMetrics
```
violate Restful API design:
- uses POST for query, without changing the server state
- uses verb in the URL

Should be:
```
GET .../TokenToLinkDevice?args=...
GET .../SalesMetrics?args=...
```

### Query Constraints and Query Arguments 
Every query may have constraints (ex. [IQueryArguments]( https://dev.heeus.io/launchpad/#!12396)) and arguments.

Constraints are:
- order (string) - order by field
- limit (int) - limit number of records
- skip (int) skip number of records
- include (string) - include referenced objects
- keys (string) - select only some field(s)
- where (object) - filter records

Arguments are optional and are passed in `&arg=...` GET parameter.

## Paths Detailed

### Create document or record
- Description:
    - Create CDoc/WDoc/CRecord/WRecord
- URL:
    - POST `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
    - Content-type: application/json
- Body: CDoc/WDoc/CRecord/WRecord
- result non-201: [error object](#errors) is returned in the body. Possible results:
    - 400: Bad Request, e.g. Record requires sys.ParentID
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord
- result 201: current WLog offset and the new IDs
 
Example result 201:
```json
{
    "CurrentWLogOffset":114,
    "NewIDs": {
        "1":322685000131212
    }
}
```

### Read document or record
- Description:
    - Reads CDoc/WDoc/CRecord/WRecord
- URL:
    - GET `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
- result 200:
    - CDoc/WDoc/CRecord/WRecord object
- result non-200: [error object](#errors) is returned in the body. Possible results:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord

### Update document or record
- Description:
    - Updates CDoc/WDoc/CRecord/WRecord
- URL:
    - PATCH `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
    - Content-type: application/json
- Body: CDoc/WDoc/CRecord/WRecord (fields to be updated)
- result non-200: [error object](#errors) is returned in the body. Possible results:
    - 400: Bad Request, e.g. Record requires sys.ParentID
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord
- result 200: current WLog offset and the new IDs

Example Result 200:
```json
{
    "CurrentWLogOffset":114,
    "NewIDs": {
        "1":322685000131212
    }
}
```

### Deactivate document or record
- Description:
    - dactivates CDoc/WDoc/CRecord/WRecord
- URL:
    - DELETE `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}/{id}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
- result non-200: [error object](#errors) is returned in the body. Possible results:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord
- result 200: current WLog offset

Example Result 200:
```json
{
    "CurrentWLogOffset":114,
}
```

### Read from CDoc collection
- URL:
    - GET `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/docs/{pkg}.{table}`
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
- Headers:
    - Authorization: Bearer {PrincipalToken}
- Result 200: 
    - The return value is a JSON object that contains a `results` field with a JSON array that lists the objects [example](../queryprocessor/request.md)
    - When the error happens during the read, the [error](#errors) property is added in the response
- Result non-200: [error object](#errors) is returned in the body. Possible results:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
- Examples:
    - Read articles
        - `GET /api/v2/untill/airs-bp3/12313123123/untill.articles?limit=20&skip=20`

### Read from Query
- URL:
    - GET `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/queries/{pkg}.{query}`
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
    - Query function argument `&arg=...`
- Headers:
    - Authorization: Bearer {PrincipalToken}
- Result 200: 
    -  The return value is a JSON object that contains a `results` field with a JSON array that lists the objects [example](../queryprocessor/request.md), ref. [Parse API](https://docs.parseplatform.org/rest/guide/#basic-queries)
    - When the error happens during the read, the [error](#errors) property is added in the response
- Result non-200: [error object](#errors) is returned in the body. Possible results:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Query Function Not Found
- Examples:
    - Read from WLog
        - `GET /api/v2/owner/app/wsid/sys.wlog?limit=100&skip=13994`
    - Read OpenAPI app schema
        - `GET /api/v2/owner/app/wsid/sys.OpenApi`

### Read from View
- URL:
    - GET `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/views/{pkg}.{view}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
- Limitations:
    -  "where" must contain "eq" or "in" condition for PK fields
- Result 200: 
    - The return value is a JSON object that contains a results field with a JSON array that lists the objects [example](../queryprocessor/request.md)
    - When the error happens during the read, the [error](#errors) property is added in the response
- Result non-200: [error object](#errors) is returned in the body. Possible results:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: View Not Found
- Examples:
    - `GET /api/v2/untill/airs-bp3/12313123123/air.SalesMetrics?where={"Year":2024, "Month":{"$in":[1,2,3]}}`

### Execute Command
- URL
    - POST `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/commands/{pkg}.{command}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
    - Content-type: application/json
- Body:
    - Command parameter or ODoc
- Result 200:
    - application/json
    - Return Type
- Result non-200: [error object](#errors) is returned in the body. Possible results:
    - 404: Command Not Found
    - 403: Forbidden
    - 401: Unauthorized

### Create/upload a new BLOB
- URL
    - POST `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs`
- Headers: 
    - Authorization: Bearer {PrincipalToken}
    - Content-type: multipart-formdata   
- Body 
    - BLOB and metadata
- Result 201:
    - application/json
    - blobId and metadata
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 413 Payload Too Large: Uploaded file exceeds allowed size.
    - 415 Unsupported Media Type: Uploaded file type is not allowed.
    - 400 Bad Request: Invalid request format.

Example request:
 ```
 POST /api/v2/users/untill/apps/airsbp3/workspaces/12344566789/blobs HTTP/1.1
Content-Type: multipart/form-data

--boundary
Content-Disposition: form-data; name="file"; filename="image.jpg"
Content-Type: image/jpeg

<binary data here>
--boundary
Content-Disposition: form-data; name="metadata"
Content-Type: application/json

{
    "name": "Example Image",
    "tags": ["image", "example", "upload"],
    "description": "This is a sample image uploaded as a BLOB."
}
--boundary--
 ```

Example response 201:
```
{
    "blobId": "1010231232123123",
    "name": "Example Image",
    "tags": ["image", "example", "upload"],
    "description": "This is a sample image uploaded as a BLOB.",
    "contentType": "image/jpeg",
    "size": 524288,  // Size in bytes
    "url": "https://federation.example.com/api/v2/users/untill/apps/airsbp3/workspaces/12344566789/blobs/1010231232123123"
}
```

### Retrieve/download the BLOB 
- Description:
    - Retrieves the BLOB data (content) or metadata
- URL
    - GET `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`
- Headers: 
    - Authorization: Bearer {PrincipalToken}
    - Accept: application/json
        - to retrieve the metadata
    - Accept: */* (default)
        - to retrieve the BLOB data
- Result 200:
    - metadata:
        - Content-type: application/json
        - Body: the BLOB metadata
    - BLOB binary data: 
        - Content-type: {storedContentType}
        - Body: binary data
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 404 Not Found: BLOB does not exist.
    - 400 Bad Request: Invalid request format.

### Update an existing BLOB's metadata
- Description:
    - Updates the BLOB data metadata
- URL
    - PATCH `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`
- Headers: 
    - Authorization: Bearer {PrincipalToken}
    - Content-type: application/json
- Body 
    - The new metadata
- Result 200:
    - Content-type: application/json
    - Body: the BLOB metadata
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 400 Bad Request: Invalid request format.
    - 404 Not Found: BLOB does not exist.
    - 415 Unsupported Media Type: Uploaded file type is not allowed.

### Replace an existing BLOB
- Description:
    - Replaces the binary data of the BLOB and optionally metadata
- URL
    - PUT `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`
- Headers: 
    - Authorization: Bearer {PrincipalToken}
    - Content-type: multipart/form-data
- Result 200:
    - Content-type: application/json
    - Body: the BLOB metadata
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 400 Bad Request: Invalid request format.
    - 404 Not Found: BLOB does not exist.
    - 413 Payload Too Large: Uploaded file exceeds allowed size.
    - 415 Unsupported Media Type: Uploaded file type is not allowed.

### Delete the existing BLOB
- Description:
    - Deletes the BLOB and its metadata
- URL
    - DELETE `/api/v2/users/{owner}/apps/{app}/workspaces/{wsid}/blobs/{blobId}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
- Successful Result:
    - 204 No Content
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 404 Not Found: BLOB does not exist.

### Read app workspaces
- Description:
    - Returns the hierarchy of app workspaces. E.g. what we show in the [workspaces section](https://github.com/untillpro/airs-design?tab=readme-ov-file#workspaces) of the app design.
    - root workspace is WSProfile
- URL
    - GET `/api/v2/users/{owner}/apps/{app}/workspaceschemas`
- Headers:
    - Authorization: Bearer {PrincipalToken}
    - Accept: 
        - application/json
        - text/markdown
- Successful Result:
    - 200 OK - app workspaces hierarchy in the selected format
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 400 Bad Request: Invalid request format

### Get workspace schema
- Description:
    - Returns the schema of the application workspace
- URL
    - GET `/api/v2/users/{owner}/apps/{app}/workspaceschemas/{pkg}.{workspace}`
- Headers:
    - Authorization: Bearer {PrincipalToken}
    - Accept: 
        - application/json
        - text/markdown
- Successful Result:
    - 200 OK - app workspaces hierarchy in the selected format
- Errors: [error object](#errors) is returned in the body. Possible results:
    - 400 Bad Request: Invalid request format
    - 404 Not Found: Workspace not found in the app

## Errors
When HTTP Result code is not OK, then [response](https://docs.parseplatform.org/rest/guide/#response-format) is an object:
```json
{
  "code": 105,
  "error": "invalid field name: bl!ng"
}
```
In the GET operations, returning the list of objects, when the error happens during the read, the "error" property may be added in the response object, meaning that the error is happened after the transmission started

# Limitations
- sys.CUD function cannot be called directly

# Technical Design
## Router:
- redirects to api v1/v2
- for v2, based on HTTP Method:
    - GET -> QP            
        - Query Function
        - System functions for:
            - Collection of CDocs
            - View
    - POST, PUT, DELETE -> CP
        - name is CDoc/WDoc/CRecord/WRecord: exec CUD command
        - POST && name_is_command: exec this command

## Updates to Query Processor
[GET params](../queryprocessor/request.md) conversion:
- Query constraints (`order`, `limit`, `skip`, `include`, `keys` -> `sys.QueryParams`
- Query `arg` -> `sys.QueryArgs`

Example:
```bash
curl -X GET \
-H "AccessToken: ${ACCESS_TOKEN}"
--data-urlencode 'arg={"SalesMode":1,"TableNumber":100,"BillPrinter":12312312312,"SalesArea":12312312333}'

  https://air.untill.com/api/rest/untill/airs-bp/140737488486431/air.IssueLinkDeviceToken

```

## Migration to GET in Queries
Some existing components must be updated:
- Air Payouts we use Query Functions for webhooks. In this case, they should be changed to commands + projectors.

## `sys.OpenApi` query function

