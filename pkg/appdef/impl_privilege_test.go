/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_Grant(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	wsName := NewQName("test", "ws")
	docName := NewQName("test", "doc")
	viewName := NewQName("test", "view")
	cmdName := NewQName("test", "cmd")
	queryName := NewQName("test", "query")

	readerRoleName := NewQName("test", "readerRole")
	writerRoleName := NewQName("test", "writerRole")
	workerRoleName := NewQName("test", "workerRole")
	ownerRoleName := NewQName("test", "ownerRole")
	admRoleName := NewQName("test", "admRole")

	t.Run("should be ok to build application with roles and privileges", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		ws := adb.AddWorkspace(wsName)

		doc := adb.AddCDoc(docName)
		doc.AddField("field1", DataKind_int32, true)
		ws.AddType(docName)

		view := adb.AddView(viewName)
		view.Key().PartKey().AddField("pk_1", DataKind_int32)
		view.Key().ClustCols().AddField("cc_1", DataKind_string)
		view.Value().AddField("vf_1", DataKind_string, false)
		ws.AddType(viewName)

		_ = adb.AddCommand(cmdName)
		ws.AddType(cmdName)

		_ = adb.AddQuery(queryName)
		ws.AddType(queryName)

		_ = adb.AddRole(readerRoleName)
		adb.Grant(PrivilegeKinds{PrivilegeKind_Select}, []QName{docName}, []FieldName{"field1"}, readerRoleName, "grant select to doc.field1")
		adb.Grant(PrivilegeKinds{PrivilegeKind_Execute}, []QName{queryName}, nil, readerRoleName, "grant execute to query")

		_ = adb.AddRole(writerRoleName)
		adb.GrantAll([]QName{wsName}, writerRoleName, "grant all to test.ws")
		adb.GrantAll([]QName{cmdName, queryName}, writerRoleName, "grant all to all functions")

		_ = adb.AddRole(workerRoleName)
		adb.GrantAll([]QName{readerRoleName, writerRoleName}, workerRoleName, "grant reader and writer roles to worker")

		_ = adb.AddRole(ownerRoleName)
		adb.GrantAll([]QName{wsName}, ownerRoleName, "grant all workspace privileges to owner")

		_ = adb.AddRole(admRoleName)
		adb.GrantAll([]QName{wsName}, admRoleName, "grant all workspace privileges to admin")
		adb.Revoke(PrivilegeKinds{PrivilegeKind_Execute}, []QName{wsName}, admRoleName, "revoke execute on workspace from admin")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

}
