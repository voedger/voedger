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
	wsName, cfgName, docName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "cfg"), appdef.NewQName("test", "doc")

	// how to build AppDef with workspace
	{
		appDef := appdef.New()

		cfg := appDef.AddCDoc(cfgName)
		cfg.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false)

		doc := appDef.AddCDoc(docName)
		doc.
			AddField("f3", appdef.DataKind_int64, true).
			AddField("f4", appdef.DataKind_string, false)

		ws := appDef.AddWorkspace(wsName)
		ws.
			AddDef(cfgName).
			AddDef(docName)

		if a, err := appDef.Build(); err == nil {
			app = a
		} else {
			panic(err)
		}
	}

	// how to inspected builded AppDef with workspace
	{
		fmt.Printf("%d definitions\n", app.DefCount())

		// how to find def by name
		def := app.Def(wsName)
		fmt.Printf("def %q: %v\n", def.QName(), def.Kind())

		// how to cast def to workspace
		w, ok := def.(appdef.IWorkspace)
		fmt.Printf("%q is workspace: %v\n", w.QName(), ok && (w.Kind() == appdef.DefKind_Workspace))

		// how to find workspace by name
		ws := app.Workspace(wsName)
		fmt.Printf("workspace %q: %v\n", ws.QName(), ws.Kind())

		// how to inspect workspace definitions names
		defCnt := 0
		ws.Defs(func(d appdef.IDef) {
			defCnt++
			fmt.Printf("%d. Name: %q, kind: %v\n", defCnt, d.QName(), d.Kind())
		})
	}

	// Output:
	// 3 definitions
	// def "test.ws": DefKind_Workspace
	// "test.ws" is workspace: true
	// workspace "test.ws": DefKind_Workspace
	// 1. Name: "test.cfg", kind: DefKind_CDoc
	// 2. Name: "test.doc", kind: DefKind_CDoc
}
