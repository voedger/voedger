/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_filter(t *testing.T) {
	require := require.New(t)
	f := filter{}
	require.Empty(f.And(), "filter.And() should be empty")
	require.Nil(f.Not(), "filter.Not() should be nil")
	require.Empty(f.Or(), "filter.Or() should be empty")
	require.Empty(f.QNames(), "filter.QNames() should be empty")
	require.Empty(f.Tags(), "filter.Tags() should be empty")
	require.Equal(appdef.TypeKindSet{}, f.Types(), "filter.Types() should be empty")
}
