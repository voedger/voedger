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

		reader := adb.AddRole(readerRoleName)
		reader.Grant([]PrivilegeKind{PrivilegeKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, "grant select from doc & view to reader")
		reader.Grant([]PrivilegeKind{PrivilegeKind_Execute}, []QName{queryName}, nil, "grant execute query to reader")

		writer := adb.AddRole(writerRoleName)
		writer.GrantAll([]QName{docName, viewName}, "grant all on doc & view to writer")
		writer.GrantAll([]QName{cmdName, queryName}, "grant execute all functions to writer")

		worker := adb.AddRole(workerRoleName)
		worker.GrantAll([]QName{readerRoleName, writerRoleName}, "grant reader and writer roles to worker")

		owner := adb.AddRole(ownerRoleName)
		owner.GrantAll([]QName{wsName}, "grant all workspace privileges to owner")

		adm := adb.AddRole(admRoleName)
		adm.GrantAll([]QName{wsName}, "grant all workspace privileges to admin")
		adm.Revoke([]PrivilegeKind{PrivilegeKind_Execute}, []QName{wsName}, "revoke execute on workspace from admin")

		intruder := adb.AddRole(intruderRoleName)
		intruder.RevokeAll([]QName{wsName}, "revoke all workspace privileges from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check roles and privileges", func(t *testing.T) {

		checkPrivilege := func(p IPrivilege, granted bool, kinds []PrivilegeKind, on []QName, fields []FieldName, to QName) {
			require.NotNil(p)
			require.Equal(granted, p.IsGranted())
			require.Equal(!granted, p.IsRevoked())
			require.Equal(kinds, p.Kinds())
			require.EqualValues(on, p.On())
			require.Equal(fields, p.Fields())
			require.Equal(to, p.To().QName())
		}

		t.Run("should be ok to enum all app roles", func(t *testing.T) {
			rolesCount := 0
			app.Roles(func(r IRole) {
				rolesCount++
				switch rolesCount {
				case 1:
					require.Equal(admRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
								[]QName{wsName}, nil,
								admRoleName)
						case 2:
							checkPrivilege(p, false,
								[]PrivilegeKind{PrivilegeKind_Execute},
								[]QName{wsName}, nil,
								admRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				case 2:
					require.Equal(intruderRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, false,
								[]PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
								[]QName{wsName}, nil,
								intruderRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 3:
					require.Equal(ownerRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
								[]QName{wsName}, nil,
								ownerRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 4:
					require.Equal(readerRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Select},
								[]QName{docName, viewName}, []FieldName{"field1"},
								readerRoleName)
						case 2:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Execute},
								[]QName{queryName}, nil,
								readerRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				case 5:
					require.Equal(workerRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Inherits},
								[]QName{readerRoleName, writerRoleName}, nil,
								workerRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(1, privilegesCount)
				case 6:
					require.Equal(writerRoleName, r.QName())
					privilegesCount := 0
					r.Privileges(func(p IPrivilege) {
						privilegesCount++
						switch privilegesCount {
						case 1:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select},
								[]QName{docName, viewName}, nil,
								writerRoleName)
						case 2:
							checkPrivilege(p, true,
								[]PrivilegeKind{PrivilegeKind_Execute},
								[]QName{cmdName, queryName}, nil,
								writerRoleName)
						default:
							require.Fail("unexpected privilege", "privilege: %v", p)
						}
					})
					require.Equal(2, privilegesCount)
				}
			})
			require.Equal(6, rolesCount)
		})

		t.Run("should be ok to enum role privileges on objects", func(t *testing.T) {
			pp := app.Role(admRoleName).PrivilegesOn([]QName{wsName})
			require.Len(pp, 2)

			checkPrivilege(pp[0], true,
				[]PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
				[]QName{wsName}, nil,
				admRoleName)
			checkPrivilege(pp[1], false,
				[]PrivilegeKind{PrivilegeKind_Execute},
				[]QName{wsName}, nil,
				admRoleName)
		})
	})
}

func Test_AppDef_AddRoleErrors(t *testing.T) {
}
