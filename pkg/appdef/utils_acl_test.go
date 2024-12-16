/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"
)

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
