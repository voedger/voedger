/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"errors"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_enrichError(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name string
		arg  any
		args []any
		want string
	}{
		{"simple string message", "test", nil, "test error: test"},
		{"format with argument", "test %d", []any{1}, "test error: test 1"},
		{"format with arguments", "test %v %v %v", []any{1, true, map[string]int{"a": 1}}, "test error: test 1 true map[a:1]"},
		{"message (not format) and params", "test", []any{"one", 2}, "test error: test one 2"},
		{"param", 1, nil, "test error: 1"},
		{"param and params", 1, []any{"2", 3.4, nil}, "test error: 1 2 3.4 <nil>"},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			err := enrichError(testError, tt.arg, tt.args...)
			require.Error(err)
			require.Equal(tt.want, err.Error())
		})
	}
}
