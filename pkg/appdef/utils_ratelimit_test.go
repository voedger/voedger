/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateScopesFrom(t *testing.T) {
	tests := []struct {
		name  string
		args  []RateScope
		want  string
		panic bool
	}{
		{"empty", []RateScope{}, `[]`, false},
		{"RateScope_AppPartition, RateScope_User", []RateScope{RateScope_AppPartition, RateScope_User}, `[AppPartition User]`, false},
		{"deduplicate", []RateScope{RateScope_AppPartition, RateScope_AppPartition}, `[AppPartition]`, false},
		// panics
		{"RateScope_null", []RateScope{RateScope_null}, `[]`, true},
		{"out of bounds", []RateScope{RateScope_count}, `[]`, true},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				require.Panics(func() {
					RateScopesFrom(tt.args...)
				}, "RateScopesFrom(%v) should panic", tt.args)
			} else {
				got := fmt.Sprint(RateScopesFrom(tt.args...))
				require.Equal(tt.want, got, "RateScopesFrom(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestRateScopes_Contains(t *testing.T) {
	tests := []struct {
		name string
		rs   RateScopes
		s    RateScope
		want bool
	}{
		{"empty scopes", RateScopes{}, RateScope_Workspace, false},
		{"basic contains", RateScopesFrom(RateScope_Workspace, RateScope_IP), RateScope_Workspace, true},
		{"basic not contains", RateScopesFrom(RateScope_AppPartition, RateScope_User), RateScope_Workspace, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.Contains(tt.s); got != tt.want {
				t.Errorf("RateScopes(%v).Contains(%v) = %v, want %v", tt.rs, tt.s, got, tt.want)
			}
		})
	}
}

func TestRateScopeTrimString(t *testing.T) {
	tests := []struct {
		name string
		s    RateScope
		want string
	}{
		{name: "basic", s: RateScope_AppPartition, want: "AppPartition"},
		{name: "out of range", s: RateScope_count + 1, want: (RateScope_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.TrimString(); got != tt.want {
				t.Errorf("%v.(RateScope).TrimString() = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func Test_validateLimitNames(t *testing.T) {

	cmdName := NewQName("test", "cmd")
	queryName := NewQName("test", "query")
	docName := NewQName("test", "doc")
	roleName := NewQName("test", "role")

	app := func() IAppDef {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddCommand(cmdName)
		_ = adb.AddQuery(queryName)
		_ = adb.AddCDoc(docName)

		_ = adb.AddRole(roleName)

		return adb.MustBuild()
	}()

	tests := []struct {
		name         string
		names        QNames
		want         error
		wantcontains string
	}{
		{"error: empty", QNames{}, ErrMissedError, ""},
		{"error: unknown name", QNames{NewQName("test", "unknown")}, ErrNotFoundError, "test.unknown"},
		{"ok: sys.ANY", QNames{QNameANY}, nil, ""},
		{"ok: sys.AnyCommand", QNames{QNameAnyCommand}, nil, ""},
		{"error: sys.AnyExtension", QNames{QNameAnyExtension}, ErrIncompatibleError, "sys.AnyExtension"},
		{"ok: test.cmd", QNames{cmdName}, nil, ""},
		{"ok: test.cmd + test.query", QNamesFrom(cmdName, queryName), nil, ""},
		{"ok: test.cmd + sys.AnyQuery", QNamesFrom(cmdName, QNameAnyQuery), nil, ""},
		{"ok: test.doc", QNames{docName}, nil, ""},
		{"ok: test.doc + sys.AnyView", QNamesFrom(docName, QNameAnyView), nil, ""},
		{"error: test.doc + sys.AnyODoc", QNamesFrom(docName, QNameAnyODoc), ErrIncompatibleError, "sys.AnyODoc"},
		{"error: test.role", QNames{roleName}, ErrIncompatibleError, "test.role"},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateLimitNames(app, tt.names)
			if tt.want == nil {
				require.NoError(got, "validateLimitNames(app, %v) returns unexpected error %v", tt.names, got)
			} else {
				require.ErrorIs(got, tt.want, "validateLimitNames(app, %v) = %v, want %v", tt.names, got, tt.want)
				if tt.wantcontains != "" {
					require.ErrorContains(got, tt.wantcontains, "validateLimitNames(app, %v) returns %v does not contains %v", tt.names, got, tt.want)
				}
			}
		})
	}
}
