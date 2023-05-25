/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_Singleton(t *testing.T) {
	require := require.New(t)

	appDef := New()

	t.Run("must be ok to create singleton definition", func(t *testing.T) {
		def := appDef.AddCDoc(NewQName("test", "singleton"))
		require.NotNil(def)

		def.SetSingleton()
		require.True(def.Singleton())
	})

	t.Run("must be panic if not CDoc definition", func(t *testing.T) {
		def := appDef.AddWDoc(NewQName("test", "wdoc"))
		require.NotNil(def)

		require.Panics(func() { def.(ICDocBuilder).SetSingleton() })

		require.False(def.(ICDocBuilder).Singleton())
	})
}
