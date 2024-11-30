/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_GrantAndRevoke(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")
	viewName := appdef.NewQName("test", "view")
	cmdName := appdef.NewQName("test", "cmd")
	queryName := appdef.NewQName("test", "query")

	readerName := appdef.NewQName("test", "reader")
	writerName := appdef.NewQName("test", "writer")
	workerName := appdef.NewQName("test", "worker")
	ownerName := appdef.NewQName("test", "owner")
	adminName := appdef.NewQName("test", "admin")

	intruderRoleName := appdef.NewQName("test", "intruder")

	t.Run("should be ok to build application with ACL", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		view := wsb.AddView(viewName)
		view.Key().PartKey().AddField("pk_1", appdef.DataKind_int32)
		view.Key().ClustCols().AddField("cc_1", appdef.DataKind_string)
		view.Value().AddField("vf_1", appdef.DataKind_string, false)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		_ = wsb.AddRole(readerName)
		wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName, viewName),
			[]appdef.FieldName{"field1"},
			readerName,
			"grant select from doc & view to reader")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(queryName),
			nil,
			readerName,
			"grant execute query to reader")

		_ = wsb.AddRole(writerName)
		wsb.GrantAll(
			filter.QNames(docName, viewName),
			writerName,
			"grant all on doc & view to writer")
		wsb.GrantAll(
			filter.QNames(cmdName, queryName),
			writerName,
			"grant execute all functions to writer")

		_ = wsb.AddRole(workerName)
		wsb.GrantAll(
			filter.QNames(readerName, writerName),
			workerName,
			"grant reader and writer roles to worker")

		_ = wsb.AddRole(ownerName)
		wsb.GrantAll(
			filter.QNames(docName, viewName),
			ownerName)
		wsb.GrantAll(
			filter.QNames(cmdName, queryName),
			ownerName)

		_ = wsb.AddRole(adminName)
		wsb.GrantAll(
			filter.QNames(ownerName),
			adminName)
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cmdName, queryName),
			nil,
			adminName,
			"revoke execute on workspace from admin")

		_ = wsb.AddRole(intruderRoleName)
		wsb.RevokeAll(
			filter.QNames(docName, viewName),
			intruderRoleName)
		wsb.RevokeAll(
			filter.QNames(cmdName, queryName),
			intruderRoleName)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested appdef.IWithACL) {
		t.Run("should be ok to enum all ACL rules", func(t *testing.T) {
			want := []struct {
				policy    appdef.PolicyKind
				ops       []appdef.OperationKind
				flt       []appdef.QName
				fields    []appdef.FieldName
				principal appdef.QName
			}{
				// reader role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName, viewName}, []appdef.FieldName{"field1"}, readerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{queryName}, nil, readerName},
				// writer role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select}, []appdef.QName{docName, viewName}, nil, writerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, writerName},
				// worker role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, []appdef.QName{readerName, writerName}, nil, workerName},
				// owner role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select}, []appdef.QName{docName, viewName}, nil, ownerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, ownerName},
				// admin role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, []appdef.QName{ownerName}, nil, adminName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, adminName},
				// intruder role
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Select}, []appdef.QName{docName, viewName}, nil, intruderRoleName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, intruderRoleName},
			}

			cnt := 0
			for r := range tested.ACL {
				require.Less(cnt, len(want))
				t.Run(fmt.Sprintf("ACL[%d]", cnt), func(t *testing.T) {
					require.Equal(want[cnt].policy, r.Policy())
					require.Equal(want[cnt].ops, r.Ops())

					got := appdef.QNames{}
					for t := range appdef.FilterMatches(r.Filter(), r.Principal().Workspace().Types()) {
						got = append(got, t.QName())
					}
					require.EqualValues(want[cnt].flt, got)

					require.Equal(want[cnt].fields, r.Filter().Fields())
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
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		docName := appdef.NewQName("test", "doc")

		cmdName := appdef.NewQName("test", "cmd")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)

		readerName := appdef.NewQName("test", "reader")

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		t.Run("if unknown principal", func(t *testing.T) {
			unknownRole := appdef.NewQName("test", "unknown")
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Select},
					filter.QNames(docName),
					nil,
					unknownRole)
			}, require.Is(appdef.ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.GrantAll(
					filter.QNames(docName),
					unknownRole)
			}, require.Is(appdef.ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.Revoke(
					[]appdef.OperationKind{appdef.OperationKind_Select},
					filter.QNames(docName),
					nil,
					unknownRole)
			}, require.Is(appdef.ErrNotFoundError), require.Has(unknownRole))
			require.Panics(func() {
				wsb.RevokeAll(
					filter.QNames(docName),
					unknownRole)
			}, require.Is(appdef.ErrNotFoundError), require.Has(unknownRole))
		})

		_ = wsb.AddRole(readerName)

		t.Run("if invalid operations", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{},
					filter.QNames(docName),
					nil,
					readerName)
			}, require.Is(appdef.ErrMissedError))
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_null},
					filter.QNames(docName),
					nil,
					readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.Has("[null]"))
			require.Panics(func() {
				wsb.Grant([]appdef.OperationKind{appdef.OperationKind_count},
					filter.QNames(docName),
					nil,
					readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.Has("[count]"))
			require.Panics(func() {
				wsb.Grant([]appdef.OperationKind{appdef.OperationKind_Select, appdef.OperationKind_Execute},
					filter.QNames(docName),
					nil,
					readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.Has("[count]"))
			require.Panics(func() {
				wsb.Revoke(
					[]appdef.OperationKind{appdef.OperationKind_Inherits},
					filter.QNames(readerName),
					nil,
					readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("revoke"), require.Has("Inherits"))
		})

		t.Run("if operations on invalid resources", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant(
					[]OperationKind{OperationKind_Select}, []QName{NewQName("test", "unknown")}, nil, readerName)
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
