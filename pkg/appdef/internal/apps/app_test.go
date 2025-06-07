/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apps_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/internal/apps"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_NewAppDef(t *testing.T) {
	require := require.New(t)

	app := apps.NewAppDef()
	require.NotNil(app)
	var _ appdef.IAppDef = app // check interface compatibility

	adb := apps.NewAppDefBuilder(app)
	require.NotNil(adb)
	var _ appdef.IAppDefBuilder = adb // check interface compatibility

	require.Equal(app, adb.AppDef(), "should be ok to obtain AppDef(*) before build")

	t.Run("Should be ok to obtain empty app", func(t *testing.T) {
		app, err := adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})
}

func Test_AppDefBuilder_MustBuild(t *testing.T) {
	require := require.New(t)

	require.NotNil(builder.New().MustBuild(), "Should be ok if no errors in builder")

	t.Run("should panic if errors in builder", func(t *testing.T) {
		adb := builder.New()
		adb.AddWorkspace(appdef.NewQName("test", "workspace")).AddView(appdef.NewQName("test", "emptyView"))

		require.Panics(func() { _ = adb.MustBuild() },
			require.Is(appdef.ErrMissedError),
			require.Has("emptyView"),
		)
	})
}
