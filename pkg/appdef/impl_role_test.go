/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
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
		owner.GrantAll([]QName{wsName}, "grant all workspace privileges to owner")

		adm := adb.AddRole(admRoleName)
		adm.GrantAll([]QName{wsName}, "grant all workspace privileges to admin")
		adm.Revoke([]OperationKind{OperationKind_Execute}, []QName{wsName}, "revoke execute on workspace from admin")

		intruder := adb.AddRole(intruderRoleName)
		intruder.RevokeAll([]QName{wsName}, "revoke all workspace privileges from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check roles", func(t *testing.T) {

		checkPrivilege := func(p IACLRule, policy PolicyKind, kinds []OperationKind, resources []QName, fields []FieldName, to QName) {
			require.NotNil(p)
			require.Equal(policy, p.Policy())
			require.Equal(kinds, p.Ops())
			require.EqualValues(resources, p.Resources().On())
			require.Equal(fields, p.Resources().Fields())
			require.Equal(to, p.Principal().QName())
		}

		t.Run("should be ok to enum all app roles", func(t *testing.T) {
			rolesCount := 0
			app.Roles(func(r IRole) {
				rolesCount++
				switch rolesCount {
				case 1:
					require.Equal(admRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
								[]QName{wsName}, nil,
								admRoleName)
						case 2:
							checkPrivilege(p, PolicyKind_Deny,
								[]OperationKind{OperationKind_Execute},
								[]QName{wsName}, nil,
								admRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				case 2:
					require.Equal(intruderRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Deny,
								[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
								[]QName{wsName}, nil,
								intruderRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 3:
					require.Equal(ownerRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
								[]QName{wsName}, nil,
								ownerRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 4:
					require.Equal(readerRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Select},
								[]QName{docName, viewName}, []FieldName{"field1"},
								readerRoleName)
						case 2:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Execute},
								[]QName{queryName}, nil,
								readerRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				case 5:
					require.Equal(workerRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Inherits},
								[]QName{readerRoleName, writerRoleName}, nil,
								workerRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 6:
					require.Equal(writerRoleName, r.QName())
					privilegesCount := 0
					r.ACL(func(p IACLRule) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select},
								[]QName{docName, viewName}, nil,
								writerRoleName)
						case 2:
							checkPrivilege(p, PolicyKind_Allow,
								[]OperationKind{OperationKind_Execute},
								[]QName{cmdName, queryName}, nil,
								writerRoleName)
						default:
							require.Fail("unexpected ACL rule", "ACL rule: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				}
			})
			require.Equal(6, rolesCount)
		})

		t.Run("should be ok to enum role privileges on resources", func(t *testing.T) {
			pp := app.Role(admRoleName).ACLForResources([]QName{wsName})
			require.Len(pp, 2)

			checkPrivilege(pp[0], PolicyKind_Allow,
				[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
				[]QName{wsName}, nil,
				admRoleName)
			checkPrivilege(pp[1], PolicyKind_Deny,
				[]OperationKind{OperationKind_Execute},
				[]QName{wsName}, nil,
				admRoleName)
		})
	})
}

func Test_AppDef_AddRoleErrors(t *testing.T) {
}
