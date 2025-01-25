/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package acl_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
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
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Activate, appdef.OperationKind_Deactivate},
			filter.QNames(docName, viewName), nil,
			writerName,
			"revoke activate/deactivate on doc & view from writer")

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
				{appdef.PolicyKind_Allow, appdef.RecordsOperations.AsArray(), []appdef.QName{docName, viewName}, nil, writerName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Activate, appdef.OperationKind_Deactivate}, []appdef.QName{docName, viewName}, nil, writerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, writerName},
				// worker role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, []appdef.QName{readerName, writerName}, nil, workerName},
				// owner role
				{appdef.PolicyKind_Allow, appdef.RecordsOperations.AsArray(), []appdef.QName{docName, viewName}, nil, ownerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, ownerName},
				// admin role
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Inherits}, []appdef.QName{ownerName}, nil, adminName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, adminName},
				// intruder role
				{appdef.PolicyKind_Deny, appdef.RecordsOperations.AsArray(), []appdef.QName{docName, viewName}, nil, intruderRoleName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Execute}, []appdef.QName{cmdName, queryName}, nil, intruderRoleName},
			}

			cnt := 0
			for _, r := range tested.ACL() {
				require.Less(cnt, len(want))
				t.Run(fmt.Sprintf("ACL[%d]", cnt), func(t *testing.T) {
					require.Equal(want[cnt].policy, r.Policy())
					require.Equal(want[cnt].ops, r.Ops())
					for _, o := range want[cnt].ops {
						require.True(r.Op(o))
					}

					flt := appdef.QNames{}
					for _, t := range appdef.FilterMatches(r.Filter(), r.Workspace().Types()) {
						flt = append(flt, t.QName())
					}
					require.EqualValues(want[cnt].flt, flt)

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
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		docName := appdef.NewQName("test", "doc")

		cmdName := appdef.NewQName("test", "cmd")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddData(appdef.NewQName("test", "data"), appdef.DataKind_int32, appdef.NullQName)

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
					[]appdef.OperationKind{}, // <-- missed operations
					filter.QNames(docName), nil, readerName)
			}, require.Is(appdef.ErrMissedError), require.Has("operations"))
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_null}, // <-- unsupported operation
					filter.QNames(docName), nil, readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("OperationKind_null"))
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_count}, // <-- unsupported operation
					filter.QNames(docName), nil, readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("OperationKind_count"))
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Select, appdef.OperationKind_Execute}, // <-- incompatible operations
					filter.QNames(docName), nil, readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.HasAll("Select", "Execute"))
			require.Panics(func() {
				wsb.Revoke(
					[]appdef.OperationKind{appdef.OperationKind_Inherits}, // <-- unsupported operation
					filter.QNames(readerName), nil, readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.HasAll("REVOKE", "Inherits"))
		})

		t.Run("if invalid resource filter", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Select},
					nil, // <-- missed filter
					nil, readerName)
			}, require.Is(appdef.ErrMissedError), require.Has("filter"))
			require.Panics(func() {
				wsb.GrantAll(
					nil, // <-- missed filter
					readerName)
			}, require.Is(appdef.ErrMissedError), require.Has("filter"))

			require.Panics(func() {
				wsb.GrantAll(filter.WSTypes(wsName, appdef.TypeKind_ViewRecord), // <-- type not found in ws
					readerName)
			}, require.Is(appdef.ErrNotFoundError), require.HasAll("ViewRecord", wsName))

			require.Panics(func() {
				wsb.GrantAll(filter.WSTypes(wsName, appdef.TypeKind_Data), // <-- unsupported ACL
					readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("test.data"))
		})

		t.Run("if operations on invalid resources", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Select},
					filter.QNames(appdef.SysData_String), // <-- type unsupported ACL
					nil, readerName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has(appdef.SysData_String))

			require.Panics(func() {
				wsb.GrantAll(filter.QNames(docName, cmdName), // <-- mixed types
					readerName)
			}, require.Is(appdef.ErrIncompatibleError))

			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Execute},
					filter.QNames(docName), // <-- incompatible operations with type
					nil, readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.Has("Execute"), require.Has(docName))
		})

		t.Run("if operations on invalid fields", func(t *testing.T) {
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(cmdName),
					[]appdef.FieldName{"field1"}, // <-- incompatible operations with fields
					readerName)
			}, require.Is(appdef.ErrIncompatibleError), require.Has("Execute"))
			require.Panics(func() {
				wsb.Grant(
					[]appdef.OperationKind{appdef.OperationKind_Select}, filter.QNames(docName),
					[]appdef.FieldName{"unknown"}, // <-- unknown field
					readerName)
			}, require.Is(appdef.ErrNotFoundError), require.Has("unknown"))
		})
	})

	t.Run("should be validate errors", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsName := appdef.NewQName("test", "workspace")
		docName := appdef.NewQName("test", "doc")

		cmdName := appdef.NewQName("test", "cmd")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)

		readerName := appdef.NewQName("test", "reader")

		doc := wsb.AddCDoc(docName)
		doc.AddField("field1", appdef.DataKind_int32, true)

		_ = wsb.AddRole(readerName)

		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(appdef.NewQName("test", "unknown")), // <-- unknown type
			nil, readerName)

		_, err := adb.Build()
		require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
	})
}

func Test_ACLWithFields(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	creatorName := appdef.NewQName("test", "creator")
	writerName := appdef.NewQName("test", "writer")
	readerName := appdef.NewQName("test", "reader")

	t.Run("should be ok to build application with ACL with fields", func(t *testing.T) {
		//         | creator | writer | reader
		//---------+---------+--------+--------
		// field_i | Insert  |   --   |   --    // #2747{Test plan}
		// field_u |   --    | Update |   --
		// field_s |   --    | Update | Select
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddCDoc(docName)
		doc.
			AddField("field_i", appdef.DataKind_int32, true).
			AddField("field_u", appdef.DataKind_int32, false).
			AddField("field_s", appdef.DataKind_int32, false)

		_ = wsb.AddRole(creatorName)
		// #2747{Test plan}
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Insert},
			filter.QNames(docName),
			[]appdef.FieldName{"field_i"},
			creatorName,
			`GRANT [Insert] ON QNAMES(test.doc)[field_i] TO test.creator`)

		_ = wsb.AddRole(writerName)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(docName),
			nil,
			writerName,
			`GRANT [Update] ON QNAMES(test.doc) TO test.writer`)
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(docName),
			[]appdef.FieldName{"field_i"},
			writerName,
			`REVOKE [Update] ON QNAMES(test.doc)[field_i] FROM test.writer`)

		_ = wsb.AddRole(readerName)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName),
			[]appdef.FieldName{"field_s"},
			readerName,
			`GRANT [Select] ON QNAMES(test.doc)[field_s] TO test.reader`)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	testWith := func(tested appdef.IWithACL) {
		t.Run("should be ok to check ACL", func(t *testing.T) {
			want := []struct {
				policy    appdef.PolicyKind
				ops       []appdef.OperationKind
				flt       []appdef.QName
				fields    []appdef.FieldName
				principal appdef.QName
			}{
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Insert}, []appdef.QName{docName}, []appdef.FieldName{"field_i"}, creatorName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Update}, []appdef.QName{docName}, nil, writerName},
				{appdef.PolicyKind_Deny, []appdef.OperationKind{appdef.OperationKind_Update}, []appdef.QName{docName}, []appdef.FieldName{"field_i"}, writerName},
				{appdef.PolicyKind_Allow, []appdef.OperationKind{appdef.OperationKind_Select}, []appdef.QName{docName}, []appdef.FieldName{"field_s"}, readerName},
			}

			cnt := 0
			for _, r := range tested.ACL() {
				require.Less(cnt, len(want))
				t.Run(fmt.Sprintf("ACL[%d]", cnt), func(t *testing.T) {
					require.Equal(want[cnt].policy, r.Policy())
					require.Equal(want[cnt].ops, r.Ops())
					for _, o := range want[cnt].ops {
						require.True(r.Op(o))
					}

					flt := appdef.QNames{}
					for _, t := range appdef.FilterMatches(r.Filter(), r.Workspace().Types()) {
						flt = append(flt, t.QName())
					}
					require.EqualValues(want[cnt].flt, flt)

					require.Equal(want[cnt].fields, r.Filter().Fields())

					require.Equal(want[cnt].principal, r.Principal().QName())

					require.Equal(r.Comment(), fmt.Sprint(r))
				})
				cnt++
			}
			require.Len(want, cnt)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
