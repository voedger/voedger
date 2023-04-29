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

	view := amock.NewView(appdef.NewQName("test", "view"))
	view.
		AddPartField("pkFld", appdef.DataKind_int64).
		AddClustColumn("ccFld", appdef.DataKind_string).
		AddValueField("vFld1", appdef.DataKind_int64, true).
		AddValueField("vFld2", appdef.DataKind_string, false)

  obj := amock.NewDef(appdef.NewQName("test", "obj"), appdef.DefKind_Object,
		amock.NewField("f1", appdef.DataKind_int64, true),
	)

	appDef := amock.NewAppDef()
	appDef.AddView(view)
	appDef.Add(obj)

  …

```
