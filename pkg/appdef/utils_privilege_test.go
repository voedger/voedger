/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivilegeKindsFrom(t *testing.T) {
	tests := []struct {
		name string
		kk   []PrivilegeKind
		want PrivilegeKinds
	}{
		{"empty", []PrivilegeKind{}, PrivilegeKinds{}},
		{"basic", []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}},
		{"remove dupes", []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Insert}, PrivilegeKinds{PrivilegeKind_Insert}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrivilegeKindsFrom(tt.kk...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PrivilegeKindsFrom() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("test panics", func(t *testing.T) {
		tests := []struct {
			name string
			kk   []PrivilegeKind
			want error
		}{
			{"null", []PrivilegeKind{PrivilegeKind_null}, ErrOutOfBoundsError},
			{"out of bounds", []PrivilegeKind{PrivilegeKind_count}, ErrOutOfBoundsError},
		}
		require := require.New(t)
		for _, tt := range tests {
			require.Panics(func() { _ = PrivilegeKindsFrom(tt.kk...) }, tt.name)
		}
	})
}

func TestPrivilegeKinds_Contains(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		k    PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, PrivilegeKind_Insert, false},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKind_Insert, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, PrivilegeKind_Select, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.Contains(tt.k); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).Contains(%v) = %v, want %v", tt.pk, tt.k, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKinds_ContainsAll(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		kk   []PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, []PrivilegeKind{PrivilegeKind_Insert}, false},
		{"empty args", PrivilegeKinds{}, []PrivilegeKind{}, true},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Select}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.ContainsAll(tt.kk...); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).ContainsAll(%v) = %v, want %v", tt.pk, tt.kk, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKinds_ContainsAny(t *testing.T) {
	tests := []struct {
		name string
		pk   PrivilegeKinds
		kk   []PrivilegeKind
		want bool
	}{
		{"empty kinds", PrivilegeKinds{}, []PrivilegeKind{PrivilegeKind_Insert}, false},
		{"empty args", PrivilegeKinds{}, []PrivilegeKind{}, true},
		{"basic contains", PrivilegeKinds{PrivilegeKind_Insert}, []PrivilegeKind{PrivilegeKind_Insert, PrivilegeKind_Update}, true},
		{"basic not contains", PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Select}, []PrivilegeKind{PrivilegeKind_Update}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pk.ContainsAny(tt.kk...); got != tt.want {
				t.Errorf("PrivilegeKinds(%v).ContainsAny(%v) = %v, want %v", tt.pk, tt.kk, got, tt.want)
			}
		})
	}
}

type mockType struct {
	IType
	kind TypeKind
	name QName
}

func (m mockType) Kind() TypeKind { return m.kind }
func (m mockType) QName() QName   { return m.name }

func TestAllPrivilegesOnType(t *testing.T) {

	testName := NewQName("test", "test")

	type args struct {
		kind TypeKind
		name QName
	}
	tests := []struct {
		name   string
		typ    args
		wantPk PrivilegeKinds
	}{
		{"null", args{TypeKind_null, NullQName},
			nil},
		{"Any", args{TypeKind_Any, QNameANY},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute, PrivilegeKind_Inherits}},
		{"Any record", args{TypeKind_Any, QNameAnyRecord},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"Any command", args{TypeKind_Any, QNameAnyCommand},
			PrivilegeKinds{PrivilegeKind_Execute}},
		{"GRecord", args{TypeKind_GRecord, testName},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"CDoc", args{TypeKind_CDoc, testName},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"View", args{TypeKind_ViewRecord, testName},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}},
		{"Command", args{TypeKind_Command, testName},
			PrivilegeKinds{PrivilegeKind_Execute}},
		{"Workspace", args{TypeKind_Workspace, testName},
			PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute}},
		{"Role", args{TypeKind_Role, testName},
			PrivilegeKinds{PrivilegeKind_Inherits}},
		{"Projector", args{TypeKind_Projector, testName},
			nil},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			typ := new(mockType)
			typ.kind = tt.typ.kind
			typ.name = tt.typ.name
			if gotPk := AllPrivilegesOnType(typ); !reflect.DeepEqual(gotPk, tt.wantPk) {
				t.Errorf("AllPrivilegesOnType(%s) = %v, want %v", tt.typ.kind.TrimString(), gotPk, tt.wantPk)
			}
		})
	}
}

func Test_validatePrivilegeOnNames(t *testing.T) {

	cdoc := NewQName("test", "cdoc")
	gdoc := NewQName("test", "gdoc")
	cmd := NewQName("test", "cmd")
	query := NewQName("test", "query")
	role := NewQName("test", "role")
	ws := NewQName("test", "ws")

	app := func() IAppDef {
		adb := New()

		_ = adb.AddCDoc(cdoc)
		_ = adb.AddGDoc(gdoc)
		_ = adb.AddCommand(cmd)
		_ = adb.AddQuery(query)
		_ = adb.AddRole(role)
		_ = adb.AddWorkspace(ws)

		return adb.MustBuild()
	}()

	tests := []struct {
		name    string
		on      []QName
		want    QNames
		wantErr error
	}{
		{"error: empty names", []QName{}, nil, ErrMissedError},
		{"error: unknown name", []QName{NewQName("test", "unknown")}, nil, ErrNotFoundError},

		{"ok: sys.ANY", []QName{QNameANY}, QNamesFrom(QNameANY), nil},
		{"error: sys.ANY + test.cmd", []QName{QNameANY, cmd}, nil, ErrIncompatibleError},

		{"ok: test.cdoc + test.gdoc", []QName{cdoc, gdoc}, QNamesFrom(cdoc, gdoc), nil},
		{"ok: sys.AnyStruct", []QName{QNameAnyStructure}, QNamesFrom(QNameAnyStructure), nil},
		{"ok: sys.AnyCDoc + test.gdoc", []QName{QNameAnyCDoc, gdoc}, QNamesFrom(QNameAnyCDoc, gdoc), nil},

		{"ok: test.cmd + test.query", []QName{cmd, query}, QNamesFrom(cmd, query), nil},
		{"ok: sys.AnyFunction", []QName{QNameAnyFunction}, QNamesFrom(QNameAnyFunction), nil},
		{"ok: sys.AnyCommand + test.query", []QName{QNameAnyCommand, query}, QNamesFrom(QNameAnyCommand, query), nil},

		{"ok test.role", []QName{role}, QNamesFrom(role), nil},
		{"error: test.role + test.cmd", []QName{role, cmd}, nil, ErrIncompatibleError},

		{"ok: test.ws", []QName{ws}, QNamesFrom(ws), nil},
		{"err: test.ws + test.cdoc", []QName{ws, cdoc}, nil, ErrIncompatibleError},

		{"error: test.cdoc + test.cmd", []QName{cdoc, cmd}, nil, ErrIncompatibleError},
		{"error: sys.AnyView + test.role", []QName{QNameAnyView, role}, nil, ErrIncompatibleError},

		{"error: sys.AnyExtension", []QName{QNameAnyExtension}, nil, ErrIncompatibleError},
	}

	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePrivilegeOnNames(app, tt.on...)
			if tt.wantErr == nil {
				require.NoError(err, "unexpected error %v in validatePrivilegeOnNames(%v)", err, tt.on)
				require.Equal(tt.want, got, "validatePrivilegeOnNames(%v): want %v, got %v", tt.on, tt.want, got)
			} else {
				require.ErrorIs(err, tt.wantErr, "expected error %v in validatePrivilegeOnNames(%v)", tt.wantErr, tt.on)
			}
		})
	}
}

func TestPrivilegeAccessControlString(t *testing.T) {
	tests := []struct {
		name  string
		grant bool
		want  string
	}{
		{"granted", true, "grant"},
		{"revoked", false, "revoke"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrivilegeAccessControlString(tt.grant); got != tt.want {
				t.Errorf("PrivilegeAccessControlString(%v) = %v, want %v", tt.grant, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    PrivilegeKind
		want string
	}{
		{name: "basic", k: PrivilegeKind_Update, want: "Update"},
		{name: "out of range", k: PrivilegeKind_count + 1, want: (PrivilegeKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(PrivilegeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
