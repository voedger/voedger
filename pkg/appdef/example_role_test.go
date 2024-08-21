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
	intruderRoleName := appdef.NewQName("test", "intruderRole")

	// how to build AppDef with roles
	{
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		ws := adb.AddWorkspace(wsName)
		ws.AddType(docName)

		reader := adb.AddRole(readerRoleName)
		reader.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName}, []appdef.FieldName{"field1"}, "grant select on doc.field1")

		writer := adb.AddRole(writerRoleName)
		writer.GrantAll([]appdef.QName{wsName}, "grant all on test.ws")

		adm := adb.AddRole(admRoleName)
		adm.GrantAll([]appdef.QName{readerRoleName, writerRoleName}, "grant reader and writer roles to adm")

		intruder := adb.AddRole(intruderRoleName)
		intruder.RevokeAll([]appdef.QName{wsName}, "revoke all on test.ws")

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
		reader.ACL(func(r appdef.IACLRule) { fmt.Println("-", r) })

		writer := app.Role(writerRoleName)
		fmt.Println(writer, ":")
		writer.ACL(func(r appdef.IACLRule) { fmt.Println("-", r) })

		adm := app.Role(admRoleName)
		fmt.Println(adm, ":")
		adm.ACL(func(r appdef.IACLRule) { fmt.Println("-", r) })

		intruder := app.Role(intruderRoleName)
		fmt.Println(intruder, ":")
		intruder.ACL(func(r appdef.IACLRule) { fmt.Println("-", r) })

		fmt.Println("ACL with select operation on test.doc:")
		for _, r := range app.ACLForResources([]appdef.QName{docName}, appdef.OperationKind_Select) {
			fmt.Println("-", r)
		}
	}

	// Output:
	// 1 Role «test.admRole»
	// 2 Role «test.intruderRole»
	// 3 Role «test.readerRole»
	// 4 Role «test.writerRole»
	// overall: 4
	// Role «test.readerRole» :
	// - grant [Select] on [test.doc]([field1]) to Role «test.readerRole»
	// Role «test.writerRole» :
	// - grant [Insert Update Select Execute] on [test.ws] to Role «test.writerRole»
	// Role «test.admRole» :
	// - grant [Inherits] on [test.readerRole test.writerRole] to Role «test.admRole»
	// Role «test.intruderRole» :
	// - revoke [Insert Update Select Execute] on [test.ws] from Role «test.intruderRole»
	// ACL with select operation on test.doc:
	// - grant [Select] on [test.doc]([field1]) to Role «test.readerRole»
}
