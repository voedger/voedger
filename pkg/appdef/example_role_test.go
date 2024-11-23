/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleRoles() {

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

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		reader := wsb.AddRole(readerRoleName)
		reader.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName}, []appdef.FieldName{"field1"}, "grant select on doc.field1")

		writer := wsb.AddRole(writerRoleName)
		writer.GrantAll([]appdef.QName{docName}, "grant all on test.doc")

		adm := wsb.AddRole(admRoleName)
		adm.GrantAll([]appdef.QName{readerRoleName, writerRoleName}, "grant reader and writer roles to adm")

		intruder := wsb.AddRole(intruderRoleName)
		intruder.RevokeAll([]appdef.QName{docName}, "revoke all on test.doc")

		app = adb.MustBuild()
	}

	// how to enum roles
	{
		cnt := 0
		for r := range appdef.Roles(app.Types) {
			cnt++
			fmt.Println(cnt, r)
		}
		fmt.Println("overall:", cnt)
	}

	// how to inspect builded AppDef with roles
	{
		reader := appdef.Role(app.Type, readerRoleName)
		fmt.Println(reader, ":")
		for r := range reader.ACL {
			fmt.Println("-", r)
		}

		writer := appdef.Role(app.Type, writerRoleName)
		fmt.Println(writer, ":")
		for r := range writer.ACL {
			fmt.Println("-", r)
		}

		adm := appdef.Role(app.Type, admRoleName)
		fmt.Println(adm, ":")
		for r := range adm.ACL {
			fmt.Println("-", r)
		}

		intruder := appdef.Role(app.Type, intruderRoleName)
		fmt.Println(intruder, ":")
		for r := range intruder.ACL {
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
	// - grant [Insert Update Select] on [test.doc] to Role «test.writerRole»
	// Role «test.admRole» :
	// - grant [Inherits] on [test.readerRole test.writerRole] to Role «test.admRole»
	// Role «test.intruderRole» :
	// - revoke [Insert Update Select] on [test.doc] from Role «test.intruderRole»
}
