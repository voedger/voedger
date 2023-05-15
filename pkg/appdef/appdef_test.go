/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddStruct(t *testing.T) {
	require := require.New(t)

	app := newAppDef()

	t.Run("panic if name is empty", func(t *testing.T) {
		require.Panics(func() {
			app.AddStruct(NullQName, DefKind_CDoc)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		require.Panics(func() {
			app.AddStruct(NewQName("naked", "🔫"), DefKind_CDoc)
		})
	})

	t.Run("panic if definition with name already exists", func(t *testing.T) {
		testName := NewQName("test", "test")
		app.AddStruct(testName, DefKind_CDoc)
		require.Panics(func() {
			app.AddStruct(testName, DefKind_CDoc)
		})
	})

	t.Run("panic if kind is not structure", func(t *testing.T) {
		require.Panics(func() { app.AddStruct(NewQName("test", "view"), DefKind_ViewRecord_Value) })
	})
}
