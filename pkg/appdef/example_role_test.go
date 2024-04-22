/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleIAppDefBuilder_AddRole() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	docName := appdef.NewQName("test", "doc")

	readerRoleName := appdef.NewQName("test", "readerRole")
	writerRoleName := appdef.NewQName("test", "writerRole")
	admRoleName := appdef.NewQName("test", "admRole")

	// how to build AppDef with roles
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		ws := adb.AddWorkspace(wsName)
		ws.AddType(docName)

		reader := adb.AddRole(readerRoleName)
		reader.Grant(appdef.PrivilegeKinds{appdef.PrivilegeKind_Select}, []appdef.QName{docName}, []appdef.FieldName{"field1"}, "grant select to doc.field1")

		writer := adb.AddRole(writerRoleName)
		writer.GrantAll([]appdef.QName{wsName}, "grant all to test.ws")

		adm := adb.AddRole(admRoleName)
		adm.GrantAll([]appdef.QName{readerRoleName, writerRoleName}, "grant reader and writer roles to adm")

		app = adb.MustBuild()
	}

	// how to enum roles
	{
		cnt := 0
		app.Roles(func(r appdef.IRole) {
			cnt++
			fmt.Println(cnt, r)
		})
		fmt.Println("overall:", cnt)
	}

	// how to inspect builded AppDef with roles
	{
		reader := app.Role(readerRoleName)
		fmt.Println(reader, ":")
		reader.Privileges(func(g appdef.IPrivilege) { fmt.Println("-", g) })

		writer := app.Role(writerRoleName)
		fmt.Println(writer, ":")
		writer.Privileges(func(g appdef.IPrivilege) { fmt.Println("-", g) })

		adm := app.Role(admRoleName)
		fmt.Println(adm, ":")
		adm.Privileges(func(g appdef.IPrivilege) { fmt.Println("-", g) })
	}

	// Output:
	// 1 Role «test.admRole»
	// 2 Role «test.readerRole»
	// 3 Role «test.writerRole»
	// overall: 3
	// Role «test.readerRole» :
	// - grant Select on [test.doc] to Role «test.readerRole»
	// Role «test.writerRole» :
	// - grant Insert on [test.ws] to Role «test.writerRole»
	// - grant Update on [test.ws] to Role «test.writerRole»
	// - grant Select on [test.ws] to Role «test.writerRole»
	// - grant Execute on [test.ws] to Role «test.writerRole»
	// Role «test.admRole» :
	// - grant Role on [test.readerRole test.writerRole] to Role «test.admRole»
}
