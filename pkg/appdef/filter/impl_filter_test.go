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
	for range f.And() {
		require.Fail("filter.And() should be empty")
	}
	require.Nil(f.Not(), "filter.Not() should be nil")
	for range f.Or() {
		require.Fail("filter.Or() should be empty")
	}
	for range f.QNames() {
		require.Fail("filter.QNames() should be empty")
	}
	for range f.Tags() {
		require.Fail("filter.Tags() should be empty")
	}
	for range f.Types() {
		require.Fail("filter.Types() should be empty")
	}
	require.Equal(appdef.NullQName, f.WS(), "filter.WS() should be NullQName")
}
