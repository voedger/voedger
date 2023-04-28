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

  pkDef := amock.NewDef(testViewRecordPkQName, appdef.DefKind_ViewRecord_PartitionKey,
		amock.NewField("pkFld", appdef.DataKind_int64, true),
	)
  ccDef := amock.NewDef(testViewRecordCcQName, appdef.DefKind_ViewRecord_ClusteringCols,
		amock.NewField("ccFld", appdef.DataKind_string, true),
	)
	valueDef := amock.NewDef(testViewRecordVQName, appdef.DefKind_ViewRecord_Value,
		amock.NewField("vFld1", appdef.DataKind_int64, true),
		amock.NewField("vFld2", appdef.DataKind_string, false),
	)
	viewDef := smock.NewDef(testViewRecordQName, appdef.DefKind_ViewRecord)
	viewDef.MockContainers(
		amock.NewContainer(appdef.SystemContainer_ViewPartitionKey, testViewRecordPkQName, 1, 1),
    amock.NewContainer(appdef.SystemContainer_ViewClusteringColumns, testViewRecordCcQName, 1, 1),
		amock.NewContainer(appdef.SystemContainer_ViewValue, testViewRecordVQName, 1, 1),
	)

	appDef := amock.NewAppDef(
		viewDef,
		pkDef,
    ccDef,
		valueDef,
	)

  …

```
