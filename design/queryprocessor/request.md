# Request example
Wit references, embedded collection
```bash
curl -X GET \
-H "AccessToken: ${ACCESS_TOKEN}"
--data-urlencode 'order=name'
--data-urlencode 'limit=10'
--data-urlencode 'skip=30'
--data-urlencode 'include=article_prices'
--data-urlencode 'refs={"dname":"id_department.name","pname":"article_prices.id_price.name"}' 
--data-urlencode 'where={"id_department":123456,"number":{"$gte": 100, "$lte": 200}}'

  https://air.untill.com/api/rest/untill/airs-bp/140737488486431/untill.articles
```

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

# See Also
- [Parse API](https://docs.parseplatform.org/rest/guide/#queries)
- [Schemas: Describe Heeus Functions with OpenAPI Standard](https://dev.heeus.io/launchpad/#!19069)
- [API v2](https://dev.heeus.io/launchpad/#!23905)

