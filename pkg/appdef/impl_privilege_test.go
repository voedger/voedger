/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_GrantAndRevoke(t *testing.T) {
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

		_ = adb.AddRole(readerRoleName)
		adb.Grant(PrivilegeKinds{PrivilegeKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerRoleName, "grant select from doc & view to reader")
		adb.Grant(PrivilegeKinds{PrivilegeKind_Execute}, []QName{queryName}, nil, readerRoleName, "grant execute query to reader")

		_ = adb.AddRole(writerRoleName)
		adb.GrantAll([]QName{docName, viewName}, writerRoleName, "grant all on doc & view to writer")
		adb.GrantAll([]QName{cmdName, queryName}, writerRoleName, "grant execute all functions to writer")

		_ = adb.AddRole(workerRoleName)
		adb.GrantAll([]QName{readerRoleName, writerRoleName}, workerRoleName, "grant reader and writer roles to worker")

		_ = adb.AddRole(ownerRoleName)
		adb.GrantAll([]QName{wsName}, ownerRoleName, "grant all workspace privileges to owner")

		_ = adb.AddRole(admRoleName)
		adb.GrantAll([]QName{wsName}, admRoleName, "grant all workspace privileges to admin")
		adb.Revoke(PrivilegeKinds{PrivilegeKind_Execute}, []QName{wsName}, admRoleName, "revoke execute on workspace from admin")

		_ = adb.AddRole(intruderRoleName)
		adb.RevokeAll([]QName{wsName}, intruderRoleName, "revoke all workspace privileges from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check roles and privileges", func(t *testing.T) {

		checkPrivilege := func(p IPrivilege, granted bool, kinds PrivilegeKinds, on QNames, fields []FieldName, to QName) {
			require.NotNil(p)
			require.Equal(granted, p.IsGranted())
			require.Equal(!granted, p.IsRevoked())
			require.Equal(kinds, p.Kinds())
			require.Equal(on, p.On())
			require.Equal(fields, p.Fields())
			require.Equal(to, p.To().QName())
		}

		t.Run("should be ok to enum all app roles", func(t *testing.T) {
			cnt := 0
			app.Privileges(func(p IPrivilege) {
				cnt++
				switch cnt {
				case 1:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Select},
						QNames{docName, viewName}, []FieldName{"field1"},
						readerRoleName)
				case 2:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Execute},
						QNames{queryName}, nil,
						readerRoleName)
				case 3:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select},
						QNames{docName, viewName}, nil,
						writerRoleName)
				case 4:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Execute},
						QNames{cmdName, queryName}, nil,
						writerRoleName)
				case 5:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Inherits},
						QNames{readerRoleName, writerRoleName}, nil,
						workerRoleName)
				case 6:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
						QNames{wsName}, nil,
						ownerRoleName)
				case 7:
					checkPrivilege(p, true,
						PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
						QNames{wsName}, nil,
						admRoleName)
				case 8:
					checkPrivilege(p, false,
						PrivilegeKinds{PrivilegeKind_Execute},
						QNames{wsName}, nil,
						admRoleName)
				case 9:
					checkPrivilege(p, false,
						PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
						QNames{wsName}, nil,
						intruderRoleName)
				default:
					require.Fail("unexpected privilege", "privilege: %v", p)
				}
			})
			require.Equal(9, cnt)
		})

		t.Run("should be ok to enum privileges on objects", func(t *testing.T) {

			t.Run("should be ok to enum privileges on ws", func(t *testing.T) {
				pp := app.PrivilegesOn(wsName)
				require.Len(pp, 4)

				checkPrivilege(pp[0], true,
					PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
					QNames{wsName}, nil,
					ownerRoleName)

				checkPrivilege(pp[1], true,
					PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
					QNames{wsName}, nil,
					admRoleName)
				checkPrivilege(pp[2], false,
					PrivilegeKinds{PrivilegeKind_Execute},
					QNames{wsName}, nil,
					admRoleName)

				checkPrivilege(pp[3], false,
					PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute},
					QNames{wsName}, nil,
					intruderRoleName)
			})

			t.Run("should be ok to enum privileges select document", func(t *testing.T) {
				pp := app.PrivilegesOn(docName, PrivilegeKind_Select)
				require.Len(pp, 2)

				checkPrivilege(pp[0], true,
					PrivilegeKinds{PrivilegeKind_Select},
					QNames{docName, viewName}, []FieldName{"field1"},
					readerRoleName)

				checkPrivilege(pp[1], true,
					PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select},
					QNames{docName, viewName}, nil,
					writerRoleName)
			})
		})
	})
}
