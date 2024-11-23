/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

		wsb := adb.AddWorkspace(NewQName("test", "workspace"))

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)
		_ = wsb.AddCDoc(docName)

		_ = wsb.AddRole(roleName)

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
			got := validateLimitNames(app.Type, tt.names)
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
