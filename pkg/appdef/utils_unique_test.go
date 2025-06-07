/*
 * Copyright (c) 2025-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_UniqueQName(t *testing.T) {
	tests := []struct {
		d    string
		n    string
		want string
	}{
		{"test.table", "unique", "test.table$uniques$unique"},
	}

	require := require.New(t)
	for _, tt := range tests {
		require.Equal(
			appdef.MustParseQName(tt.want),
			appdef.UniqueQName(appdef.MustParseQName(tt.d), tt.n))
	}
}
