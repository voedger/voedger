/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
)

func ExampleIWithACL() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	docName := appdef.NewQName("test", "doc")

	readerRoleName := appdef.NewQName("test", "readerRole")
	writerRoleName := appdef.NewQName("test", "writerRole")
	admRoleName := appdef.NewQName("test", "admRole")
	intruderRoleName := appdef.NewQName("test", "intruderRole")

	// how to build AppDef with roles
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		_ = wsb.AddRole(readerRoleName)
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Select}, filter.QNames(docName), []appdef.FieldName{"field1"}, readerRoleName, "grant select on doc.field1 to reader")

		_ = wsb.AddRole(writerRoleName)
		wsb.GrantAll(filter.QNames(docName), writerRoleName, "grant all on test.doc to writer")

		_ = wsb.AddRole(admRoleName)
		wsb.GrantAll(filter.QNames(readerRoleName, writerRoleName), admRoleName, "grant reader and writer roles to adm")

		_ = wsb.AddRole(intruderRoleName)
		wsb.RevokeAll(filter.QNames(docName), intruderRoleName, "revoke all on test.doc from intruder")

		app = adb.MustBuild()
	}

	// how to enum roles
	{
		cnt := 0
		for r := range appdef.Roles(app.Types()) {
			cnt++
			fmt.Println(cnt, r)
		}
		fmt.Println("overall:", cnt)
	}

	// how to inspect ACL rules
	{
		ws := app.Workspace(wsName)
		fmt.Println("ACL in", ws, ":")
		for i, acl := range ws.ACL() {
			fmt.Println(i+1, acl, ":", acl.Comment())
		}
		fmt.Println("overall:", len(ws.ACL()))
	}

	// Output:
	// 1 Role «test.admRole»
	// 2 Role «test.intruderRole»
	// 3 Role «test.readerRole»
	// 4 Role «test.writerRole»
	// overall: 4
	// ACL in Workspace «test.ws» :
	// 1 GRANT [Select] ON QNAMES(test.doc)[field1] TO test.readerRole : grant select on doc.field1 to reader
	// 2 GRANT [Insert Update Activate Deactivate Select] ON QNAMES(test.doc) TO test.writerRole : grant all on test.doc to writer
	// 3 GRANT [Inherits] ON QNAMES(test.readerRole, test.writerRole) TO test.admRole : grant reader and writer roles to adm
	// 4 REVOKE [Insert Update Activate Deactivate Select] ON QNAMES(test.doc) FROM test.intruderRole : revoke all on test.doc from intruder
	// overall: 4
}
