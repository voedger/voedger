[![codecov](https://codecov.io/gh/heeus/schemas/branch/main/graph/badge.svg?token=u6VrbqKtnn)](https://codecov.io/gh/heeus/schemas/mock)

# schemas/mock

Useful for test schemas.

``` golang
  import (
    …
    smock "github.com/voedger/voedger/pkg/schemas/mock"
    …
  )

  …
  pkSchema := smock.MockedSchema(testViewRecordPkQName, appdef.SchemaKind_ViewRecord_PartitionKey,
		smock.MockedField("pkFld", appdef.DataKind_int64, true),
	)
  ccSchema := smock.MockedSchema(testViewRecordCcQName, appdef.SchemaKind_ViewRecord_ClusteringCols,
		smock.MockedField("ccFld", appdef.DataKind_string, true),
	)
	valueSchema := smock.MockedSchema(testViewRecordVQName, appdef.SchemaKind_ViewRecord_Value,
		smock.MockedField("vFld1", appdef.DataKind_int64, true),
		smock.MockedField("vFld2", appdef.DataKind_string, false),
	)
	viewSchema := smock.MockedSchema(testViewRecordQName1, appdef.SchemaKind_ViewRecord)
	viewSchema.MockContainers(
		smock.MockedContainer(appdef.SystemContainer_ViewPartitionKey, testViewRecordPkQName, 1, 1),
    smock.MockedContainer(appdef.SystemContainer_ViewClusteringColumns, testViewRecordCcQName, 1, 1),
		smock.MockedContainer(appdef.SystemContainer_ViewValue, testViewRecordVQName, 1, 1),
	)

	cache := smock.MockedSchemaCache(
		viewSchema,
		pkSchema,
    ccSchema,
		valueSchema,
	)

  …

```
