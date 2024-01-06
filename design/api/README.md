# Motivation
- We need good REST API for Voedger
- Old API must still be available until the new one is fully developed, so we can continue with AIR

# Functional Design

## API URL
API URL must support versioning ([example IBM MQ](https://www.ibm.com/docs/en/ibm-mq/9.1?topic=api-rest-versions), [example Chargebee](https://apidocs.chargebee.com/docs/api/)):

- old API is available at `/api/v1/...` (for the period of AIR migration it will be available both on `/api/` and `/api/v1/`)
- new API is available at `/api/v2/...`
    - "v1" is not allowed as an owner name, at least until API "v1" is ready

## REST API Paths

| Action                               | REST API Path                                  |
|--------------------------------------|------------------------------------------------|
| Create CDoc/WDoc/CRecord/WRecord     | `POST /api/v2/owner/app/wsid/pkg.table`        |
| Read CDoc/WDoc/CRecord/WRecord       | `GET /api/v2/owner/app/wsid/pkg.table/id`      |
| Update CDoc/WDoc/CRecord/WRecord     | `PUT /api/v2/owner/app/wsid/pkg.table/id`      |
| Deactivate CDoc/WDoc/CRecord/WRecord | `DELETE /api/v2/owner/app/wsid/pkg.table/id`   |
| Execute Command                      | `POST /api/v2/owner/app/wsid/pkg.command`      |
| Execute Query (old way*)             | `POST /api/v2/owner/app/wsid/pkg.name`         |
|   - Read Collection                  |   - `POST /api/v2/owner/app/wsid/pkg.table`    |
|   - Execute Query Function           |   - `POST /api/v2/owner/app/wsid/pkg.query`    |
| Execute Query (new way*)             | `GET /api/v2/owner/app/wsid/pkg.name`          |
|   - Read Query Function              |   - `GET /api/v2/owner/app/wsid/pkg.query`     |
|   - Read Collection                  |   - `GET /api/v2/owner/app/wsid/pkg.table`     |

* Current design of the QueryProcessor based on POST queries. If change it to GET (new way), then we do not need prefixes like 'q.' and 'c.' for commands and queries in URLs:

| HTTP Method       | Processor         |
|-------------------|-------------------|
| GET               | Query Processor   |
| POST, PUT, DELETE | Command Processor |

## Paths Detailed

### Create CDoc/WDoc/CRecord/WRecord object

- URL:
    - `POST /api/v2/owner/app/wsid/pkg.table`
- Parameters:
    - application/json
    - CDoc/WDoc/CRecord/WRecord
- Result:
    - application/json
    - ID of the new object
        - ??? entire CDoc/WDoc/CRecord/WRecord object
- Errors:
    - 400: Bad Request, e.g. Record requires sys.ParentID
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord

### Read CDoc/WDoc/CRecord/WRecord
- URL:
    - `GET /api/v2/owner/app/wsid/pkg.table/id`
- Parameters: none
- Result:
    - application/json
    - CDoc/WDoc/CRecord/WRecord object
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord

### Update CDoc/WDoc/CRecord/WRecord
- URL:
    - `PUT /api/v2/owner/app/wsid/pkg.table/id`
- Parameters: 
    - application/json
    - CDoc/WDoc/CRecord/WRecord (fields to be updated)
- Result: none
    - ??? entire CDoc/WDoc/CRecord/WRecord object
- Errors:
    - 400: Bad Request, e.g. Record requires sys.ParentID
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord

### Deactivate CDoc/WDoc/CRecord/WRecord
- URL:
    - `DELETE /api/v2/owner/app/wsid/pkg.table/id`
- Parameters: none
- Result: none
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
    - 405: Method Not Allowed, table is an ODoc/ORecord

### Execute Query (old way)
- URL:
    - `POST /api/v2/owner/app/wsid/pkg.name`:
        - `POST /api/v2/owner/app/wsid/pkg.query` - Query Function
        - `POST /api/v2/owner/app/wsid/pkg.table` - Collection
- Parameters: according to [current QP syntax](https://dev.heeus.io/launchpad/#!18998)
- Result:
    - multi-dimentional array
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Query Function or Table Not Found

### Execute Query (new way)
- URL:
    - `GET /api/v2/owner/app/wsid/pkg.name`
        - `GET /api/v2/owner/app/wsid/pkg.query`
        - `GET /api/v2/owner/app/wsid/pkg.table`
- Parameters: [example](../queryprocessor/request.md), request syntax taken from [Parse API](https://docs.parseplatform.org/rest/guide/#queries), 
- Result: one of:
    - `application/json` array of objects, [example](../queryprocessor/request.md)
    - `application/json` an object
    - `mime/type` custom type of response
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Query Function or Table Not Found

Examples:
- Read from WLog
    - `GET /api/v2/owner/app/wsid/sys.wlog?limit=100&skip=13994`
- Read articles
    - `GET /api/v2/untill/airs-bp3/12313123123/untill.articles?limit=20&skip=20`
- Read OpenAPI app schema
    - `GET /api/v2/owner/app/wsid/sys.OpenApi`


### Execute Command
- URL
    - `POST /api/v2/owner/app/wsid/pkg.command`
- Parameters: 
    - application/json
    - Parameter Type / ODoc
- Result:
    - application/json
    - Return Type
- Errors:
    - 404: Command Not Found
    - 403: Forbidden
    - 401: Unauthorized

# Limitations
- sys.CUD function cannot be called directly

# Technical Design
- Router:
    - redirects to api v1/v2
    - for v2, based on HTTP Method:
        - GET -> QP            
        - POST, PUT, DELETE -> CP
            - name is CDoc/WDoc/CRecord/WRecord: exec CUD command
            - POST && name_is_command: exec this command
- Updates to Query Processor to support v2
- `sys.OpenApi` query function
