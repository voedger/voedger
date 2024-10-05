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
		adb.GrantAll([]QName{docName, viewName}, ownerRoleName)
		adb.GrantAll([]QName{cmdName, queryName}, ownerRoleName)

		_ = adb.AddRole(admRoleName)
		adb.GrantAll([]QName{ownerRoleName}, admRoleName)
		adb.Revoke([]OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, admRoleName, "revoke execute on workspace from admin")

		_ = adb.AddRole(intruderRoleName)
		adb.RevokeAll([]QName{docName, viewName}, intruderRoleName)
		adb.RevokeAll([]QName{cmdName, queryName}, intruderRoleName)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to enum all ACL rules", func(t *testing.T) {
		wantACL := []struct {
			policy    PolicyKind
			ops       []OperationKind
			res       []QName
			fields    []FieldName
			principal QName
		}{
			// reader role
			{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName, viewName}, []FieldName{"field1"}, readerRoleName},
			{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{queryName}, nil, readerRoleName},
			// writer role
			{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, writerRoleName},
			{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, writerRoleName},
			// worker role
			{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{readerRoleName, writerRoleName}, nil, workerRoleName},
			// owner role
			{PolicyKind_Allow, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, ownerRoleName},
			{PolicyKind_Allow, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, ownerRoleName},
			// admin role
			{PolicyKind_Allow, []OperationKind{OperationKind_Inherits}, []QName{ownerRoleName}, nil, admRoleName},
			{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, admRoleName},
			// intruder role
			{PolicyKind_Deny, []OperationKind{OperationKind_Insert, OperationKind_Update, OperationKind_Select}, []QName{docName, viewName}, nil, intruderRoleName},
			{PolicyKind_Deny, []OperationKind{OperationKind_Execute}, []QName{cmdName, queryName}, nil, intruderRoleName},
		}

		aclCount := 0
		for r := range app.ACL {
			require.Less(aclCount, len(wantACL))
			t.Run(fmt.Sprintf("ACL[%d]", aclCount), func(t *testing.T) {
				require.Equal(wantACL[aclCount].policy, r.Policy())
				require.Equal(wantACL[aclCount].ops, r.Ops())
				require.EqualValues(wantACL[aclCount].res, r.Resources().On())
				require.Equal(wantACL[aclCount].fields, r.Resources().Fields())
				require.Equal(wantACL[aclCount].principal, r.Principal().QName())
			})
			aclCount++
		}
		require.Len(wantACL, aclCount)
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
				adb.Revoke([]OperationKind{OperationKind_Select}, []QName{docName}, nil, unknownRole)
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
			require.Panics(func() {
				adb.Revoke([]OperationKind{OperationKind_Inherits}, []QName{readerRoleName}, nil, readerRoleName)
			}, require.Is(ErrUnsupportedError), require.Has("revoke"), require.Has("Inherits"))
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
		adb.Grant([]OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"field1"}, readerRoleName, "grant select field1 from doc to reader")

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to check ACL", func(t *testing.T) {
		wantACL := []struct {
			policy    PolicyKind
			ops       []OperationKind
			res       []QName
			fields    []FieldName
			principal QName
		}{
			{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName}, nil, readerRoleName},
			{PolicyKind_Allow, []OperationKind{OperationKind_Select}, []QName{docName}, []FieldName{"field1"}, readerRoleName},
		}

		aclCount := 0
		for r := range app.ACL {
			require.Less(aclCount, len(wantACL))
			t.Run(fmt.Sprintf("ACL[%d]", aclCount), func(t *testing.T) {
				require.Equal(wantACL[aclCount].policy, r.Policy())
				require.Equal(wantACL[aclCount].ops, r.Ops())
				require.EqualValues(wantACL[aclCount].res, r.Resources().On())
				require.Equal(wantACL[aclCount].fields, r.Resources().Fields())
				require.Equal(wantACL[aclCount].principal, r.Principal().QName())
			})
			aclCount++
		}
		require.Len(wantACL, aclCount)
	})
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
			k:    PolicyKind_Count + 1,
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
