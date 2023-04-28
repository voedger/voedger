/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_Add(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("panic if name is empty", func(t *testing.T) {
		require.Panics(func() {
			app.Add(NullQName, DefKind_CDoc)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		require.Panics(func() {
			app.Add(NewQName("naked", "🔫"), DefKind_CDoc)
		})
	})

	t.Run("if schema with name already exists", func(t *testing.T) {
		testName := NewQName("test", "test")
		app.Add(testName, DefKind_CDoc)
		require.Panics(func() {
			app.Add(testName, DefKind_CDoc)
		})
	})
}
