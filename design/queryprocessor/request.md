# Motivation
Support [Parse API](https://docs.parseplatform.org/rest/guide/#queries) request syntax

# Request example
Wit references, embedded collection
```bash
curl -X GET \
-H "AccessToken: ${ACCESS_TOKEN}"
--data-urlencode 'order=name'
--data-urlencode 'limit=10'
--data-urlencode 'skip=30'
--data-urlencode 'include=article_prices.article_price_exceptions'  #include article_prices and article_price_exceptions
--data-urlencode 'select=name,article_prices.price,article_prices.article_price_exceptions.name'
--data-urlencode 'where={"id_department":123456,"number":{"$gte": 100, "$lte": 200}}'

  https://air.untill.com/api/rest/untill/airs-bp/140737488486431/untill.articles
```


Parameters:
- order (string) - order by field
- limit (int) - limit number of records
- skip (int) skip number of records
- include (string) - include referenced objects
- keys (string) - select only some field(s)
- where (object) - filter records

# Response
```json
[
    {
        "id": 1234,
        "number": 100,
        "name": "article1",
        "id_department": 123456,
        "__refs": {
            "dname": "Cold Drinks"
        }, 
        "article_prices": [
            {
                "id": 321,
                "id_price": 3214,
                "price": 12.23,
                "__refs": {
                    "pname": "Normal Price"
                }
            }
        ]
    }
]
```

# Links
- [Parse API](https://docs.parseplatform.org/rest/guide/#queries)
- [Parse API: select only some fields](http://parseplatform.org/Parse-SDK-JS/api/3.4.2/Parse.Query.html#select)
    - [see also: Stack overflow](https://stackoverflow.com/questions/61100282/parse-server-select-a-few-fields-from-included-object)
- [Launchpad: Schemas: Describe Heeus Functions with OpenAPI Standard](https://dev.heeus.io/launchpad/#!19069)
- [Launchpad: API v2](https://dev.heeus.io/launchpad/#!23905)


# Misc

Parse API [multi-level inclding](https://docs.parseplatform.org/rest/guide/#relational-queries):
```bash
curl -X GET \
  -H "X-Parse-Application-Id: ${APPLICATION_ID}" \
  -H "X-Parse-REST-API-Key: ${REST_API_KEY}" \
  -G \
  --data-urlencode 'order=-createdAt' \
  --data-urlencode 'limit=10' \
  --data-urlencode 'include=post.author' \
  https://YOUR.PARSE-SERVER.HERE/parse/classes/Comment
``````