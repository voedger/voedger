/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_GrantAndRevoke(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	wsName := NewQName("test", "workspace")
	docName := NewQName("test", "doc")
	viewName := NewQName("test", "view")
	cmdName := NewQName("test", "cmd")
	queryName := NewQName("test", "query")

	readerName := NewQName("test", "reader")
	writerName := NewQName("test", "writer")
	workerName := NewQName("test", "worker")
	ownerName := NewQName("test", "owner")
	adminName := NewQName("test", "admin")

	intruderRoleName := NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", DataKind_int32, true)

		view := wsb.AddView(viewName)
		view.Key().PartKey().AddField("pk_1", DataKind_int32)
		view.Key().ClustCols().AddField("cc_1", DataKind_string)
		view.Value().AddField("vf_1", DataKind_string, false)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		_ = wsb.AddRole(readerName)
		wsb.Grant([]OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerName, "grant select from doc & view to reader")
		wsb.Grant([]OperationKind{OperationKind_Execute}, []QName{queryName}, nil, readerName, "grant execute query to reader")

		_ = wsb.AddRole(writerName)
		wsb.GrantAll([]QName{docName, viewName}, writerName, "grant all on doc & view to writer")
		wsb.GrantAll([]QName{cmdName, queryName}, writerName, "grant execute all functions to writer")

		_ = wsb.AddRole(workerName)
		wsb.GrantAll([]QName{readerName, writerName}, workerName, "grant reader and writer roles to worker")

		_ = wsb.AddRole(ownerName)
		wsb.GrantAll([]QName{docName, viewName}, ownerName)
		wsb.GrantAll([]QName{cmdName, queryName}, ownerName)

		_ = wsb.AddRole(adminName)
		wsb.GrantAll([]QName{ownerName}, adminName)
		wsb.Revoke([]OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, adminName, "revoke execute on workspace from admin")

		_ = wsb.AddRole(intruderRoleName)
		wsb.RevokeAll([]QName{docName, viewName}, intruderRoleName)
		wsb.RevokeAll([]QName{cmdName, queryName}, intruderRoleName)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested IWithACL) {
		t.Run("should be ok to enum all ACL rules", func(t *testing.T) {
			want := []struct {
				policy    PolicyKind
				ops       []OperationKind
				res       []QName
				fields    []FieldName
				principal QName
			}{
				// reader role
				{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{queryName}, nil, readerName},
				// writer role
				{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, writerName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, writerName},
				// worker role
				{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{readerName, writerName}, nil, workerName},
				// owner role
				{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, ownerName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, ownerName},
				// admin role
				{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{ownerName}, nil, adminName},
				{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, adminName},
				// intruder role
				{PolicyKind_Deny, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, intruderRoleName},
				{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, intruderRoleName},
			}

			cnt := 0
			for r := range tested.ACL {
				require.Less(cnt, len(want))
				t.Run(fmt.Sprintf("ACL[%d]", cnt), func(t *testing.T) {
					require.Equal(want[cnt].policy, r.Policy())
					require.Equal(want[cnt].ops, r.Ops())
					require.EqualValues(want[cnt].res, r.Resources().On())
					require.Equal(want[cnt].fields, r.Resources().Fields())
					require.Equal(want[cnt].principal, r.Principal().QName())
				})
				cnt++
			}
			require.Len(want, cnt)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func Test_GrantAndRevokeErrors(t *testing.T) {
	require := require.New(t)
	t.Run("should be panics", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsName := NewQName("test", "workspace")
		docName := NewQName("test", "doc")

		cmdName := NewQName("test", "cmd")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)

		readerName := NewQName("test", "reader")

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", DataKind_int32, true)

		t.Run("if unknown principal", func(t *testing.T) {
			unknownRole := NewQName("test", "unknown")
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, nil, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.GrantAll([]QName{docName}, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.Revoke([]OperationKind{OperationKind_Select}, []QName{docName}, nil, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.RevokeAll([]QName{docName}, unknownRole)
			}, require.Is(ErrNotFoundError), require.Has(unknownRole))
		})

		_ = wsb.AddRole(readerName)

		t.Run("if invalid operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant([]OperationKind{}, []QName{docName}, nil, readerName)
			}, require.Is(ErrMissedError))
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_null}, []QName{docName}, nil, readerName)
			}, require.Is(ErrIncompatibleError), require.Has("[null]"))
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_count}, []QName{docName}, nil, readerName)
			}, require.Is(ErrIncompatibleError), require.Has("[count]"))
			require.Panics(func() {
				wsb.Revoke([]OperationKind{OperationKind_Inherits}, []QName{readerName}, nil, readerName)
			}, require.Is(ErrUnsupportedError), require.Has("revoke"), require.Has("Inherits"))
		})

		t.Run("if operations on invalid resources", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Select}, []QName{}, nil, readerName)
			}, require.Is(ErrMissedError))
			require.Panics(func() {
				wsb.GrantAll(nil, readerName)
			}, require.Is(ErrMissedError))

			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Select}, []QName{NewQName("test", "unknown")}, nil, readerName)
			}, require.Is(ErrNotFoundError), require.Has("test.unknown"))

			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Select}, []QName{SysData_String}, nil, readerName)
			}, require.Is(ErrIncompatibleError), require.Has(SysData_String))

			require.Panics(func() {
				wsb.GrantAll([]QName{docName, wsName}, readerName)
			}, require.Is(ErrIncompatibleError))

			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Execute}, []QName{docName}, nil, readerName)
			}, require.Is(ErrIncompatibleError), require.Has("Execute"), require.Has(docName))
		})

		t.Run("if operations on invalid fields", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Execute}, []QName{cmdName}, []FieldName{"field1"}, readerName)
			}, require.Is(ErrIncompatibleError), require.Has("Execute"))
			require.Panics(func() {
				wsb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"unknown"}, readerName)
			}, require.Is(ErrNotFoundError), require.Has("unknown"))
		})
	})
}

func Test_ACLWithFields(t *testing.T) {
	require := require.New(t)

	var app IAppDef

	wsName := NewQName("test", "workspace")
	docName := NewQName("test", "doc")

	creatorName := NewQName("test", "creator")
	writerName := NewQName("test", "writer")
	readerName := NewQName("test", "reader")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		//         | creator | writer | reader
		//---------+---------+--------+--------
		// field_i | Insert  |   --   |   --    // #2747{Test plan}
		// field_u |   --    | Update |   --
		// field_s |   --    | Update | Select
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.
			AddField("field_i", DataKind_int32, true).
			AddField("field_u", DataKind_int32, false).
			AddField("field_s", DataKind_int32, false)

		wsb.AddRole(creatorName).
			// #2747{Test plan}
			Grant([]OperationKind{OperationKind_Insert}, []QName{docName}, []FieldName{"field_i"},
				`GRANT INSERT test.doc(field_i) TO test.creator`)
		wsb.AddRole(writerName).
			Grant([]OperationKind{OperationKind_Update}, []QName{docName}, nil,
				`GRANT UPDATE test.doc TO test.writer`).
			Revoke([]OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"field_i"},
				`REVOKE UPDATE test.doc(field_i) FROM test.writer`)
		wsb.AddRole(readerName).
			Grant([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"field_s"},
				`GRANT SELECT test.doc(field_s) TO test.reader`)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested IWithACL) {
		t.Run("should be ok to check ACL", func(t *testing.T) {
			want := []struct {
				policy    PolicyKind
				ops       []OperationKind
				res       []QName
				fields    []FieldName
				principal QName
			}{
				{PolicyKind_Allow, []OperationKind{OperationKind_Insert}, []QName{docName}, []FieldName{"field_i"}, creatorName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Update}, []QName{docName}, nil, writerName},
				{PolicyKind_Deny, []OperationKind{OperationKind_Update}, []QName{docName}, []FieldName{"field_i"}, writerName},
				{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"field_s"}, readerName},
			}

			cnt := 0
			for r := range tested.ACL {
				require.Less(cnt, len(want))
				t.Run(fmt.Sprintf("ACL[%d]", cnt), func(t *testing.T) {
					require.Equal(want[cnt].policy, r.Policy())
					require.Equal(want[cnt].ops, r.Ops())
					require.EqualValues(want[cnt].res, r.Resources().On())
					require.Equal(want[cnt].fields, r.Resources().Fields())
					require.Equal(want[cnt].principal, r.Principal().QName())
				})
				cnt++
			}
			require.Len(want, cnt)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func TestPolicyKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    PolicyKind
		want string
	}{
		{
			name: "0 —> `PolicyKind_null`",
			k:    PolicyKind_null,
			want: `PolicyKind_null`,
		},
		{
			name: "1 —> `PolicyKind_Allow`",
			k:    PolicyKind_Allow,
			want: `PolicyKind_Allow`,
		},
		{
			name: "4 —> `PolicyKind(4)`",
			k:    PolicyKind_count + 1,
			want: `PolicyKind(4)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.String(); got != tt.want {
				t.Errorf("PolicyKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
