/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_FilterMatches(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	cmdName := appdef.NewQName("test", "cmd")
	queryName := appdef.NewQName("test", "query")

	t.Run("should be ok to build application", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to filter matches", func(t *testing.T) {
		filtered := appdef.FilterMatches(filter.AllFunctions(wsName), app.Types())
		cnt := 0
		for t := range filtered {
			switch cnt {
			case 0:
				require.Equal(cmdName, t.QName())
			case 1:
				require.Equal(queryName, t.QName())
			default:
				require.Fail("unexpected type", "type: %v", t)
			}
			cnt++
		}
		require.Equal(2, cnt)

		t.Run("filter matches iter should be breakable", func(t *testing.T) {
			cnt := 0
			for range filtered {
				cnt++
				break
			}
			require.Equal(1, cnt)
		})
	})
}
