/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
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

	t.Run("should be ok to build application with ACL", func(t *testing.T) {
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
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerRoleName, "grant select from doc & view to reader")
		adb.Grant([]OperationKind{OperationKind_Execute}, []QName{queryName}, nil, readerRoleName, "grant execute query to reader")

		_ = adb.AddRole(writerRoleName)
		adb.GrantAll([]QName{docName, viewName}, writerRoleName, "grant all on doc & view to writer")
		adb.GrantAll([]QName{cmdName, queryName}, writerRoleName, "grant execute all functions to writer")

		_ = adb.AddRole(workerRoleName)
		adb.GrantAll([]QName{readerRoleName, writerRoleName}, workerRoleName, "grant reader and writer roles to worker")

		_ = adb.AddRole(ownerRoleName)
		adb.GrantAll([]QName{wsName}, ownerRoleName, "grant all workspace operations to owner")

		_ = adb.AddRole(admRoleName)
		adb.GrantAll([]QName{wsName}, admRoleName, "grant all workspace operations to admin")
		adb.Revoke([]OperationKind{OperationKind_Execute}, []QName{wsName}, admRoleName, "revoke execute on workspace from admin")

		_ = adb.AddRole(intruderRoleName)
		adb.RevokeAll([]QName{wsName}, intruderRoleName, "revoke all workspace operations from intruder")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check ACL", func(t *testing.T) {

		checkACLRule := func(acl IACLRule, policy PolicyKind, kinds []OperationKind, resources []QName, fields []FieldName, principal QName) {
			require.NotNil(acl)
			require.Equal(policy, acl.Policy())
			require.Equal(kinds, acl.Ops())
			require.EqualValues(resources, acl.Resources().On())
			require.Equal(fields, acl.Resources().Fields())
			require.Equal(principal, acl.Principal().QName())
		}

		t.Run("should be ok to enum all ACL rules", func(t *testing.T) {
			cnt := 0
			app.ACL(func(p IACLRule) {
				cnt++
				switch cnt {
				case 1:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Select},
						[]QName{docName, viewName}, []FieldName{"field1"},
						readerRoleName)
				case 2:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Execute},
						[]QName{queryName}, nil,
						readerRoleName)
				case 3:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select},
						[]QName{docName, viewName}, nil,
						writerRoleName)
				case 4:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Execute},
						[]QName{cmdName, queryName}, nil,
						writerRoleName)
				case 5:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Inherits},
						[]QName{readerRoleName, writerRoleName}, nil,
						workerRoleName)
				case 6:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
						[]QName{wsName}, nil,
						ownerRoleName)
				case 7:
					checkACLRule(p, PolicyKind_Allow,
						[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
						[]QName{wsName}, nil,
						admRoleName)
				case 8:
					checkACLRule(p, PolicyKind_Deny,
						[]OperationKind{OperationKind_Execute},
						[]QName{wsName}, nil,
						admRoleName)
				case 9:
					checkACLRule(p, PolicyKind_Deny,
						[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
						[]QName{wsName}, nil,
						intruderRoleName)
				default:
					require.Fail("unexpected ACL Rule", "ACL rule: %v", p)
				}
			})
			require.Equal(9, cnt)
		})

		t.Run("should be ok to enum ACL for resource", func(t *testing.T) {

			t.Run("should be ok to enum ACL for ws", func(t *testing.T) {
				pp := app.ACLForResources([]QName{wsName})
				require.Len(pp, 4)

				checkACLRule(pp[0], PolicyKind_Allow,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					ownerRoleName)

				checkACLRule(pp[1], PolicyKind_Allow,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					admRoleName)
				checkACLRule(pp[2], PolicyKind_Deny,
					[]OperationKind{OperationKind_Execute},
					[]QName{wsName}, nil,
					admRoleName)

				checkACLRule(pp[3], PolicyKind_Deny,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					intruderRoleName)
			})

			t.Run("should be ok to enum all ACL with select", func(t *testing.T) {
				pp := app.ACLForResources([]QName{}, OperationKind_Select)
				require.Len(pp, 5)

				checkACLRule(pp[0], PolicyKind_Allow,
					[]OperationKind{OperationKind_Select},
					[]QName{docName, viewName}, []FieldName{"field1"},
					readerRoleName)

				checkACLRule(pp[1], PolicyKind_Allow,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select},
					[]QName{docName, viewName}, nil,
					writerRoleName)

				checkACLRule(pp[2], PolicyKind_Allow,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					ownerRoleName)

				checkACLRule(pp[3], PolicyKind_Allow,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					admRoleName)

				checkACLRule(pp[4], PolicyKind_Deny,
					[]OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute},
					[]QName{wsName}, nil,
					intruderRoleName)
			})
		})
	})
}

func Test_AppDef_GrantAndRevokeErrors(t *testing.T) {
	require := require.New(t)
	t.Run("panics while to build application with ACL", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsName := NewQName("test", "ws")
		docName := NewQName("test", "doc")

		cmdName := NewQName("test", "cmd")
		_ = adb.AddCommand(cmdName)

		readerRoleName := NewQName("test", "readerRole")

		_ = adb.AddWorkspace(wsName)

		doc := adb.AddCDoc(docName)
		doc.AddField("field1", DataKind_int32, true)

		t.Run("should be panic if unknown principal", func(t *testing.T) {
			unknownRole := NewQName("test", "unknownRole")
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, nil, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				adb.GrantAll([]QName{docName}, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				adb.Revoke([]OperationKind{OperationKind_Select}, []QName{docName}, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				adb.RevokeAll([]QName{docName}, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
		})

		_ = adb.AddRole(readerRoleName)

		t.Run("should be panic if invalid operations", func(t *testing.T) {
			require.Panics(func() {
				adb.Grant([]OperationKind{}, []QName{docName}, nil, readerRoleName)
			}, require.Is(ErrMissedError))
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_null}, []QName{docName}, nil, readerRoleName)
			}, require.Is(ErrIncompatibleError), require.Has("[null]"))
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_count}, []QName{docName}, nil, readerRoleName)
			}, require.Is(ErrIncompatibleError), require.Has("[count]"))
		})

		t.Run("should be panic if operations on invalid resources", func(t *testing.T) {
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Select}, []QName{}, nil, readerRoleName)
			}, require.Is(ErrMissedError))
			require.Panics(func() {
				adb.GrantAll(nil, readerRoleName)
			}, require.Is(ErrMissedError))

			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Select}, []QName{NewQName("test", "unknown")}, nil, readerRoleName)
			}, require.Is(ErrNotFoundError), require.Has("test.unknown"))

			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Select}, []QName{SysData_String}, nil, readerRoleName)
			}, require.Is(ErrIncompatibleError), require.Has(SysData_String))

			require.Panics(func() {
				adb.GrantAll([]QName{docName, wsName}, readerRoleName)
			}, require.Is(ErrIncompatibleError))

			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Execute}, []QName{docName}, nil, readerRoleName)
			}, require.Is(ErrIncompatibleError), require.Has("Execute"), require.Has(docName))
		})

		t.Run("should be panic if operations on invalid fields", func(t *testing.T) {
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Execute}, []QName{cmdName}, []FieldName{"field1"}, readerRoleName)
			}, require.Is(ErrIncompatibleError), require.Has("Execute"))
			require.Panics(func() {
				adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"unknown"}, readerRoleName)
			}, require.Is(ErrNotFoundError), require.Has("unknown"))
		})
	})
}

func Test_AppDef_GrantWithFields(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	docName := NewQName("test", "doc")

	readerRoleName := NewQName("test", "readerRole")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		doc := adb.AddCDoc(docName)
		doc.AddField("field1", DataKind_int32, true)

		_ = adb.AddRole(readerRoleName)
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, nil, readerRoleName, "grant select any field from doc to reader")
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{QNameAnyStructure}, []FieldName{"field1"}, readerRoleName, "grant select field1 from any to reader")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check ACL", func(t *testing.T) {

		checkRule := func(p IACLRule, policy PolicyKind, kinds []OperationKind, resources []QName, fields []FieldName, to QName) {
			require.NotNil(p)
			require.Equal(policy, p.Policy())
			require.Equal(kinds, p.Ops())
			require.EqualValues(resources, p.Resources().On())
			require.Equal(fields, p.Resources().Fields())
			require.Equal(to, p.Principal().QName())
		}

		cnt := 0
		app.ACL(func(p IACLRule) {
			cnt++
			switch cnt {
			case 1:
				checkRule(p, PolicyKind_Allow,
					[]OperationKind{OperationKind_Select},
					[]QName{docName}, nil,
					readerRoleName)
			case 2:
				checkRule(p, PolicyKind_Allow,
					[]OperationKind{OperationKind_Select},
					[]QName{QNameAnyStructure}, []FieldName{"field1"},
					readerRoleName)
			default:
				require.Fail("unexpected ACL rule", "ACL rule: %v", p)
			}
		})

		require.Equal(2, cnt)
	})
}
