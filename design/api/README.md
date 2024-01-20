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

| Action                               | REST API Path                                  |
|--------------------------------------|------------------------------------------------|
| Create CDoc/WDoc/CRecord/WRecord     | `POST /api/v2/owner/app/wsid/pkg.table`        |
| Update CDoc/WDoc/CRecord/WRecord     | `PATCH /api/v2/owner/app/wsid/pkg.table/id`    |
| Deactivate CDoc/WDoc/CRecord/WRecord | `DELETE /api/v2/owner/app/wsid/pkg.table/id`   |
| Execute Command                      | `POST /api/v2/owner/app/wsid/pkg.command`      |
| Read CDoc/WDoc/CRecord/WRecord       | `GET /api/v2/owner/app/wsid/pkg.table/id`      |
| Read from Query Function             | `GET /api/v2/owner/app/wsid/pkg.query`         |
| Read from CDoc Collection            | `GET /api/v2/owner/app/wsid/pkg.table`         |
| Read from View                       | `GET /api/v2/owner/app/wsid/pkg.view`          |


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

>> Note: according to RESTful API design, queries should not change the state of the system. Current QueryFunction design allows it to execute commands through HTTP bus.

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

### ACL
EXECUTE -> SELECT for Queries?

Currently:
```sql
LIMIT AllQueriesLimit EXECUTE ON ALL QUERIES WITH TAG PosTag WITH RATE AppDefaultRate;
GRANT EXECUTE ON QUERY Query1 TO LocationUser;
```

Should be:
```sql
LIMIT AllQueriesLimit SELECT ON ALL QUERIES WITH TAG PosTag WITH RATE AppDefaultRate;
GRANT SELECT ON QUERY Query1 TO LocationUser;
```


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
    - `PATCH /api/v2/owner/app/wsid/pkg.table/id`
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

### Read from Query
- URL:
    - `GET /api/v2/owner/app/wsid/pkg.query`
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
    - Query function argument `&arg=...`
- Result: 
    -  The return value is a JSON object that contains a results field with a JSON array that lists the objects [example](../queryprocessor/request.md)
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Query Function Not Found
- Examples:
    - Read from WLog
        - `GET /api/v2/owner/app/wsid/sys.wlog?limit=100&skip=13994`
    - Read OpenAPI app schema
        - `GET /api/v2/owner/app/wsid/sys.OpenApi`

### Read from CDoc collection
- URL:
    - `GET /api/v2/owner/app/wsid/pkg.table`
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
- Result: 
    -  The return value is a JSON object that contains a results field with a JSON array that lists the objects [example](../queryprocessor/request.md)
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: Table Not Found
- Examples:
    - Read articles
        - `GET /api/v2/untill/airs-bp3/12313123123/untill.articles?limit=20&skip=20`

### Read from View
- URL:
    - `GET /api/v2/owner/app/wsid/pkg.view`
- Parameters: 
    - Query [constraints](../queryprocessor/request.md)
    - Key values in `&arg=...`:
        - example `&arg={"yyyymm":202401}`
- Result: 
    -  The return value is a JSON object that contains a results field with a JSON array that lists the objects [example](../queryprocessor/request.md)
- Errors:
    - 401: Unauthorized
    - 403: Forbidden
    - 404: View Not Found
- Examples:
    - Read articles
        - `GET /api/v2/untill/airs-bp3/12313123123/untill.articles?limit=20&skip=20`

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

## Updates to vSQL and appdef
- Query Result can be either array of objects or an object
- Query Result can be either JSON or custom mime type

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

