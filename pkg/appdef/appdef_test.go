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
			app.AddCDoc(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		require.Panics(func() {
			app.AddCDoc(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if definition with name already exists", func(t *testing.T) {
		testName := NewQName("test", "test")
		app.AddCDoc(testName)
		require.Panics(func() {
			app.AddWDoc(testName)
		})
	})
}
