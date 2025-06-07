/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package roles_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestRoles(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	docName := appdef.NewQName("test", "doc")
	viewName := appdef.NewQName("test", "view")
	cmdName := appdef.NewQName("test", "cmd")
	queryName := appdef.NewQName("test", "query")

	readerRoleName := appdef.NewQName("test", "readerRole")
	writerRoleName := appdef.NewQName("test", "writerRole")
	workerRoleName := appdef.NewQName("test", "workerRole")
	ownerRoleName := appdef.NewQName("test", "ownerRole")
	admRoleName := appdef.NewQName("test", "admRole")

	intruderRoleName := appdef.NewQName("test", "intruderRole")

	t.Run("should be ok to build application with roles", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		view := wsb.AddView(viewName)
		view.Key().PartKey().AddField("pk_1", appdef.DataKind_int32)
		view.Key().ClustCols().AddField("cc_1", appdef.DataKind_string)
		view.Value().AddField("field1", appdef.DataKind_string, false)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		_ = wsb.AddRole(readerRoleName)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName, viewName),
			[]appdef.FieldName{"field1"},
			readerRoleName,
			"grant select from doc & view to reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(queryName),
			nil,
			readerRoleName,
			"grant execute query to reader")

		_ = wsb.AddRole(writerRoleName)
		wsb.GrantAll(
			filter.QNames(docName),
			writerRoleName,
			"grant all on doc to writer")
		wsb.GrantAll(
			filter.QNames(viewName),
			writerRoleName,
			"grant all on view to writer")
		wsb.GrantAll(
			filter.QNames(cmdName, queryName),
			writerRoleName,
			"grant execute all functions to writer")

		_ = wsb.AddRole(workerRoleName)
		wsb.GrantAll(
			filter.QNames(readerRoleName, writerRoleName),
			workerRoleName,
			"grant reader and writer roles to worker")

		_ = wsb.AddRole(ownerRoleName)
		wsb.GrantAll(
			filter.QNames(docName),
			ownerRoleName,
			"grant all on doc to owner")
		wsb.GrantAll(
			filter.QNames(viewName),
			ownerRoleName,
			"grant all on view to owner")
		wsb.GrantAll(
			filter.QNames(cmdName, queryName),
			ownerRoleName,
			"grant execute all functions to owner")

		_ = wsb.AddRole(admRoleName)
		wsb.GrantAll(
			filter.QNames(ownerRoleName),
			admRoleName,
			"grant owner to admin")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cmdName, queryName),
			nil,
			admRoleName,
			"revoke execute from admin")

		_ = wsb.AddRole(intruderRoleName)
		wsb.RevokeAll(
			filter.QNames(docName),
			intruderRoleName,
			"revoke doc from intruder")
		wsb.RevokeAll(
			filter.QNames(viewName),
			intruderRoleName,
			"revoke view from intruder")
		wsb.RevokeAll(
			filter.QNames(cmdName, queryName),
			intruderRoleName,
			"revoke funcs from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested interface {
		types.IWithTypes
		appdef.IWithACL
	}) {
		t.Run("should be ok to enum roles", func(t *testing.T) {
			want := appdef.QNamesFrom(readerRoleName, writerRoleName, workerRoleName, ownerRoleName, admRoleName, intruderRoleName)
			got := appdef.QNames{}
			for r := range appdef.Roles(tested.Types()) {
				got.Add(r.QName())
			}
			require.Equal(want, got)
		})
		t.Run("should be ok to find role", func(t *testing.T) {
			r := tested.Type(workerRoleName)
			require.Equal(appdef.TypeKind_Role, r.Kind())

			role := appdef.Role(tested.Type, workerRoleName)
			require.Equal(r.(appdef.IRole), role)
			require.Equal(workerRoleName, role.QName())
			require.Equal(wsName, role.Workspace().QName())

			require.Nil(appdef.Role(tested.Type, appdef.NewQName("test", "unknown")), "should be nil if not found")
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func Test_RoleInheritanceWithComplexFilter(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")

	anc1RoleName := appdef.NewQName("test", "ancestor1Role")
	anc2RoleName := appdef.NewQName("test", "ancestor2Role")
	descRoleName := appdef.NewQName("test", "descendantRole")

	ancTag := appdef.NewQName("test", "ancestorTag")

	t.Run("should be ok to build application with complex role inheritance", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(ancTag)

		wsb.AddRole(anc1RoleName).SetTag(ancTag)
		wsb.AddRole(anc2RoleName).SetTag(ancTag)

		_ = wsb.AddRole(descRoleName)
		wsb.GrantAll(filter.Tags(ancTag), descRoleName, "grant all ancestor roles to descendant")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})
}

// #3335: Test for published role
func Test_RolePublished(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")

	roleName := appdef.NewQName("test", "role")

	t.Run("should be ok to build application with published role", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddRole(roleName).SetPublished(true)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to find published role", func(t *testing.T) {
		role := appdef.Role(app.Workspace(wsName).Type, roleName)
		require.NotNil(role)
		require.True(role.Published())
	})
}
