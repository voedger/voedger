/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
)

func ExampleIWorkspace() {

	var app appdef.IAppDef
	wsName, descName, docName, recName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "desc"), appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

	// how to build AppDef with workspace
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		ws.AddCDoc(descName).
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		ws.SetDescriptor(descName)

		cDoc := ws.AddCDoc(docName)
		cDoc.
			AddField("d1", appdef.DataKind_int64, true).
			AddField("d2", appdef.DataKind_string, false)
		cDoc.
			AddContainer("rec", recName, 0, 100)

		ws.AddCRecord(recName).
			AddField("r1", appdef.DataKind_int64, true).
			AddField("r2", appdef.DataKind_string, false)

		app = adb.MustBuild()
	}

	// how to enum workspaces
	{
		fmt.Println("App workspaces:")
		for i, ws := range app.Workspaces() {
			fmt.Println("-", i+1, ":", ws)
		}
	}

	// how to inspect workspace
	{
		// how to find workspace by name
		ws := app.Workspace(wsName)
		fmt.Println(ws)

		// how to inspect workspace
		fmt.Println("Workspace descriptor is", ws.Descriptor())
		fmt.Println("Workspace local types:")
		for i, t := range ws.LocalTypes() {
			fmt.Println("-", i+1, ":", t)
		}
	}

	// how to find workspace by descriptor
	{
		ws := app.WorkspaceByDescriptor(descName)
		fmt.Println()
		fmt.Printf("Founded by descriptor %q: %v\n", descName, ws)
	}

	// Output:
	// App workspaces:
	// - 1 : Workspace «sys.Workspace»
	// - 2 : Workspace «test.ws»
	// Workspace «test.ws»
	// Workspace descriptor is test.desc
	// Workspace local types:
	// - 1 : CDoc «test.desc»
	// - 2 : CDoc «test.doc»
	// - 3 : CRecord «test.rec»
	//
	// Founded by descriptor "test.desc": Workspace «test.ws»
}
