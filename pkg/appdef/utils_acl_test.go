/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
)

func TestPolicyKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.PolicyKind
		want string
	}{
		{
			name: "0 —> `PolicyKind_null`",
			k:    appdef.PolicyKind_null,
			want: `PolicyKind_null`,
		},
		{
			name: "1 —> `PolicyKind_Allow`",
			k:    appdef.PolicyKind_Allow,
			want: `PolicyKind_Allow`,
		},
		{
			name: "4 —> `PolicyKind(4)`",
			k:    appdef.PolicyKind_count + 1,
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
func TestPolicyKindActionString(t *testing.T) {
	tests := []struct {
		name   string
		policy appdef.PolicyKind
		want   string
	}{
		{"granted", appdef.PolicyKind_Allow, "GRANT"},
		{"revoked", appdef.PolicyKind_Deny, "REVOKE"},
		{"none", appdef.PolicyKind_null, "null"},
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
		k    appdef.OperationKind
		want string
	}{
		{name: "basic", k: appdef.OperationKind_Update, want: "Update"},
		{name: "out of range", k: appdef.OperationKind_count + 1, want: (appdef.OperationKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(PrivilegeKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
