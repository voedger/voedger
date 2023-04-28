[![codecov](https://codecov.io/gh/voedger/voedger/appdef/mock/branch/main/graph/badge.svg?token=u6VrbqKtnn)](https://codecov.io/gh/voedger/voedger/appdef/mock)

# appdef/mock

Useful for test application definition.

``` golang
  import (
    …
    "github.com/voedger/voedger/pkg/appdef"
    amock "github.com/voedger/voedger/pkg/appdef/mock"
    …
  )

  …
	testViewRecordQName, testViewRecordPkQName, testViewRecordCcQName, testViewRecordVQName :=
		appdef.NewQName("test", "view"), 
		appdef.NewQName("test", "viewPk"), 
		appdef.NewQName("test", "viewCc"), 
		appdef.NewQName("test", "viewValue")

  pkSchema := amock.NewSchema(testViewRecordPkQName, appdef.DefKind_ViewRecord_PartitionKey,
		amock.NewField("pkFld", appdef.DataKind_int64, true),
	)
  ccSchema := amock.NewSchema(testViewRecordCcQName, appdef.DefKind_ViewRecord_ClusteringCols,
		amock.NewField("ccFld", appdef.DataKind_string, true),
	)
	valueSchema := amock.NewSchema(testViewRecordVQName, appdef.DefKind_ViewRecord_Value,
		amock.NewField("vFld1", appdef.DataKind_int64, true),
		amock.NewField("vFld2", appdef.DataKind_string, false),
	)
	viewSchema := smock.NewSchema(testViewRecordQName, appdef.DefKind_ViewRecord)
	viewSchema.MockContainers(
		amock.NewContainer(appdef.SystemContainer_ViewPartitionKey, testViewRecordPkQName, 1, 1),
    amock.NewContainer(appdef.SystemContainer_ViewClusteringColumns, testViewRecordCcQName, 1, 1),
		amock.NewContainer(appdef.SystemContainer_ViewValue, testViewRecordVQName, 1, 1),
	)

	appDef := amock.NewAppDef(
		viewSchema,
		pkSchema,
    ccSchema,
		valueSchema,
	)

  …

```
