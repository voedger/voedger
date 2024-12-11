/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/set"
)

func Test_allACLOperationsOnType(t *testing.T) {

	testName := NewQName("test", "test")

	type typ struct {
		kind TypeKind
		name QName
	}
	tests := []struct {
		name   string
		typ    typ
		wantPk set.Set[OperationKind]
	}{
		{"null", typ{TypeKind_null, NullQName},
			set.Empty[OperationKind]()},
		{"GRecord", typ{TypeKind_GRecord, testName},
			set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)},
		{"CDoc", typ{TypeKind_CDoc, testName},
			set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)},
		{"View", typ{TypeKind_ViewRecord, testName},
			set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)},
		{"Command", typ{TypeKind_Command, testName},
			set.From(OperationKind_Execute)},
		{"Role", typ{TypeKind_Role, testName},
			set.From(OperationKind_Inherits)},
		{"Projector", typ{TypeKind_Projector, testName},
			set.Empty[OperationKind]()},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			typ := new(mockType)
			typ.kind = tt.typ.kind
			typ.name = tt.typ.name
			if gotPk := allACLOperationsOnType(typ); !reflect.DeepEqual(gotPk, tt.wantPk) {
				t.Errorf("allACLOperationsOnType(%s) = %v, want %v", tt.typ.kind.TrimString(), gotPk, tt.wantPk)
			}
		})
	}
}

func TestPolicyKindActionString(t *testing.T) {
	tests := []struct {
		name   string
		policy PolicyKind
		want   string
	}{
		{"granted", PolicyKind_Allow, "GRANT"},
		{"revoked", PolicyKind_Deny, "REVOKE"},
		{"none", PolicyKind_null, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.ActionString(); got != tt.want {
				t.Errorf("%v.ActionString() = %v, want %v", tt.policy, got, tt.want)
			}
		})
	}
}

func TestPrivilegeKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    OperationKind
		want string
	}{
		{name: "basic", k: OperationKind_Update, want: "Update"},
		{name: "out of range", k: OperationKind_count + 1, want: (OperationKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(PrivilegeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
