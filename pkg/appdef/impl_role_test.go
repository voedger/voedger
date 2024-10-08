/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddRole(t *testing.T) {
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

	intruderRoleName := NewQName("test", "intruderRole")

	t.Run("should be ok to build application with roles", func(t *testing.T) {
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

		reader := adb.AddRole(readerRoleName)
		reader.Grant([]OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, "grant select from doc & view to reader")
		reader.Grant([]OperationKind{OperationKind_Execute}, []QName{queryName}, nil, "grant execute query to reader")

		writer := adb.AddRole(writerRoleName)
		writer.GrantAll([]QName{docName, viewName}, "grant all on doc & view to writer")
		writer.GrantAll([]QName{cmdName, queryName}, "grant execute all functions to writer")

		worker := adb.AddRole(workerRoleName)
		worker.GrantAll([]QName{readerRoleName, writerRoleName}, "grant reader and writer roles to worker")

		owner := adb.AddRole(ownerRoleName)
		owner.GrantAll([]QName{docName, viewName}, "grant all on doc & view to owner")
		owner.GrantAll([]QName{cmdName, queryName}, "grant execute all functions to owner")

		adm := adb.AddRole(admRoleName)
		adm.GrantAll([]QName{ownerRoleName})
		adm.Revoke([]OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, "revoke execute from admin")

		intruder := adb.AddRole(intruderRoleName)
		intruder.RevokeAll([]QName{docName, viewName}, "revoke all from intruder")
		intruder.RevokeAll([]QName{cmdName, queryName}, "revoke all from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to enum all app roles", func(t *testing.T) {
		type wantACL []struct {
			policy PolicyKind
			ops    []OperationKind
			res    []QName
			fld    []FieldName
			to     QName
		}
		tt := []struct {
			name QName
			wantACL
		}{ // sorted by name
			{admRoleName, wantACL{
				{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{ownerRoleName}, nil, admRoleName},
				{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, admRoleName},
			}},
			{intruderRoleName, wantACL{
				{PolicyKind_Deny, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, intruderRoleName},
				{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, intruderRoleName},
			}},
			{ownerRoleName, wantACL{
				{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, ownerRoleName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, ownerRoleName},
			}},
			{readerRoleName, wantACL{
				{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerRoleName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{queryName}, nil, readerRoleName},
			}},
			{workerRoleName, wantACL{
				{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{readerRoleName, writerRoleName}, nil, workerRoleName},
			}},
			{writerRoleName, wantACL{
				{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, writerRoleName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, writerRoleName},
			}},
		}

		rolesCount := 0
		for r := range app.Roles {
			require.Equal(tt[rolesCount].name, r.QName())
			wantACL := tt[rolesCount].wantACL
			aclCount := 0
			for acl := range r.ACL {
				t.Run(fmt.Sprintf("%v.ACL[%d]", r, aclCount), func(t *testing.T) {
					require.Equal(wantACL[aclCount].policy, acl.Policy())
					require.Equal(wantACL[aclCount].ops, acl.Ops())
					require.EqualValues(wantACL[aclCount].res, acl.Resources().On())
					require.Equal(wantACL[aclCount].fld, acl.Resources().Fields())
					require.Equal(wantACL[aclCount].to, acl.Principal().QName())
				})
				aclCount++
			}
			require.Len(wantACL, aclCount)
			rolesCount++
		}
		require.Equal(6, rolesCount)
	})

	t.Run("role.Anc() should return inheritance", func(t *testing.T) {
		roles := app.Role(workerRoleName).AncRoles()
		require.Equal([]QName{readerRoleName, writerRoleName}, roles)
	})
}

func Test_AppDef_AddRoleErrors(t *testing.T) {
}
