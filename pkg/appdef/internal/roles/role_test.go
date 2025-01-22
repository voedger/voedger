/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package roles_test

import (
	"fmt"
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

		reader := wsb.AddRole(readerRoleName)
		reader.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName, viewName),
			[]appdef.FieldName{"field1"},
			"grant select from doc & view to reader")
		reader.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(queryName),
			nil,
			"grant execute query to reader")

		writer := wsb.AddRole(writerRoleName)
		writer.GrantAll(
			filter.QNames(docName, viewName),
			"grant all on doc & view to writer")
		writer.GrantAll(
			filter.QNames(cmdName, queryName),
			"grant execute all functions to writer")

		worker := wsb.AddRole(workerRoleName)
		worker.GrantAll(
			filter.QNames(readerRoleName, writerRoleName),
			"grant reader and writer roles to worker")

		owner := wsb.AddRole(ownerRoleName)
		owner.GrantAll(
			filter.QNames(docName, viewName),
			"grant all on doc & view to owner")
		owner.GrantAll(
			filter.QNames(cmdName, queryName),
			"grant execute all functions to owner")

		adm := wsb.AddRole(admRoleName)
		adm.GrantAll(filter.QNames(ownerRoleName))
		adm.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cmdName, queryName),
			nil,
			"revoke execute from admin")

		intruder := wsb.AddRole(intruderRoleName)
		intruder.RevokeAll(
			filter.QNames(docName, viewName),
			"revoke all from intruder")
		intruder.RevokeAll(
			filter.QNames(cmdName, queryName),
			"revoke all from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to enum roles", func(t *testing.T) {
			type wantACL []struct {
				policy appdef.PolicyKind
				ops    []appdef.OperationKind
				flt    []appdef.QName
				fld    []appdef.FieldName
				to     appdef.QName
			}
			tt := []struct {
				name appdef.QName
				wantACL
			}{ // sorted by name
				{admRoleName, wantACL{
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, appdef.QNames{ownerRoleName}, nil, admRoleName},
					{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.QNames{cmdName, queryName}, nil, admRoleName},
				}},
				{intruderRoleName, wantACL{
					{appdef.PolicyKind_Deny, appdef.RecordsOperations.AsArray(), appdef.QNames{docName, viewName}, nil, intruderRoleName},
					{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.QNames{cmdName, queryName}, nil, intruderRoleName},
				}},
				{ownerRoleName, wantACL{
					{appdef.PolicyKind_Allow, appdef.RecordsOperations.AsArray(), appdef.QNames{docName, viewName}, nil, ownerRoleName},
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.QNames{cmdName, queryName}, nil, ownerRoleName},
				}},
				{readerRoleName, wantACL{
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Select}, appdef.QNames{docName, viewName}, []appdef.FieldName{"field1"}, readerRoleName},
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.QNames{queryName}, nil, readerRoleName},
				}},
				{workerRoleName, wantACL{
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, appdef.QNames{readerRoleName, writerRoleName}, nil, workerRoleName},
				}},
				{writerRoleName, wantACL{
					{appdef.PolicyKind_Allow, appdef.RecordsOperations.AsArray(), appdef.QNames{docName, viewName}, nil, writerRoleName},
					{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.QNames{cmdName, queryName}, nil, writerRoleName},
				}},
			}

			rolesCount := 0
			for r := range appdef.Roles(tested.Types()) {
				require.Equal(tt[rolesCount].name, r.QName())
				wantACL := tt[rolesCount].wantACL
				aclCount := 0
				for _, acl := range r.ACL() {
					t.Run(fmt.Sprintf("%v.ACL[%d]", r, aclCount), func(t *testing.T) {
						require.Equal(wantACL[aclCount].policy, acl.Policy())
						require.Equal(wantACL[aclCount].ops, acl.Ops())
						for _, o := range wantACL[aclCount].ops {
							require.True(acl.Op(o))
						}

						flt := appdef.QNames{}
						for _, t := range appdef.FilterMatches(acl.Filter(), tested.Types()) {
							flt = append(flt, t.QName())
						}
						require.EqualValues(wantACL[aclCount].flt, flt)

						require.Equal(wantACL[aclCount].fld, acl.Filter().Fields())
						require.Equal(wantACL[aclCount].to, acl.Principal().QName())
					})
					aclCount++
				}
				require.Len(wantACL, aclCount)
				rolesCount++
			}
			require.Equal(6, rolesCount)
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

		wsb.AddRole(descRoleName).GrantAll(filter.Tags(ancTag))

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})
}
