/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIWorkspace() {

	var app appdef.IAppDef
	wsName, descName, docName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc"), appdef.NewQName("test", "doc")

	// how to build AppDef with workspace
	{
		appDef := appdef.New()

		appDef.AddCDoc(descName).
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		appDef.AddCDoc(docName).
			AddField("f3", appdef.DataKind_int64, true).
			AddField("f4", appdef.DataKind_string, false)

		appDef.AddWorkspace(wsName).
			SetDescriptor(descName).
			AddDef(docName)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspected builded AppDef with workspace
	{
		// how to find workspace by name
		ws := app.Workspace(wsName)
		fmt.Printf("workspace %q: %v\n", ws.QName(), ws.Kind())

		// how to inspect workspace definitions names
		fmt.Printf("workspace %q descriptor is %q\n", ws.QName(), ws.Descriptor())

		ws.Defs(func(d appdef.IDef) {
			fmt.Printf("- Def: %q, kind: %v\n", d.QName(), d.Kind())
		})
	}

	// Output:
	// workspace "test.ws": DefKind_Workspace
	// workspace "test.ws" descriptor is "test.desc"
	// - Def: "test.doc", kind: DefKind_CDoc
}
