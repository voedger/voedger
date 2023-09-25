/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AppDef_AddCommand(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	cmdName, argName, unlName, resName := NewQName("test", "cmd"), NewQName("test", "arg"), NewQName("test", "unl"), NewQName("test", "res")

	t.Run("must be ok to add command", func(t *testing.T) {
		appDef := New()

		_ = appDef.AddObject(argName)
		_ = appDef.AddObject(unlName)
		_ = appDef.AddObject(resName)

		cmd := appDef.AddCommand(cmdName)
		require.Equal(TypeKind_Command, cmd.Kind())
		require.Equal(cmd, appDef.Command(cmdName))
		require.Nil(cmd.Arg())
		require.Nil(cmd.UnloggedArg())
		require.Nil(cmd.Result())

		t.Run("must be ok to assign cmd args and result", func(t *testing.T) {
			cmd.
				SetArg(argName).
				SetUnloggedArg(unlName).
				SetResult(resName).
				SetExtension("CmdExt", ExtensionEngineKind_BuiltIn)
		})

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := appDef.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	t.Run("must be ok to find builded command", func(t *testing.T) {
		typ := app.Type(cmdName)
		require.Equal(TypeKind_Command, typ.Kind())

		c, ok := typ.(ICommand)
		require.True(ok)
		require.Equal(TypeKind_Command, c.Kind())

		cmd := app.Command(cmdName)
		require.Equal(TypeKind_Command, cmd.Kind())
		require.Equal(c, cmd)

		require.Equal(argName, cmd.Arg().QName())
		require.Equal(TypeKind_Object, cmd.Arg().Kind())

		require.Equal(unlName, cmd.UnloggedArg().QName())
		require.Equal(TypeKind_Object, cmd.UnloggedArg().Kind())

		require.Equal(resName, cmd.Result().QName())
		require.Equal(TypeKind_Object, cmd.Result().Kind())

		require.Equal("CmdExt", cmd.Extension().Name())
		require.Equal(ExtensionEngineKind_BuiltIn, cmd.Extension().Engine())
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Nil(app.Command(unknown))
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddCommand(NullQName)
		})
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		apb := New()
		require.Panics(func() {
			apb.AddCommand(NewQName("naked", "🔫"))
		})
	})

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		apb := New()
		apb.AddObject(testName)
		require.Panics(func() {
			apb.AddCommand(testName)
		})
	})

	t.Run("panic if extension name is empty", func(t *testing.T) {
		apb := New()
		cmd := apb.AddCommand(NewQName("test", "cmd"))
		require.Panics(func() {
			cmd.SetExtension("", ExtensionEngineKind_BuiltIn)
		})
	})

	t.Run("panic if extension name is invalid", func(t *testing.T) {
		apb := New()
		cmd := apb.AddCommand(NewQName("test", "cmd"))
		require.Panics(func() {
			cmd.SetExtension("naked 🔫", ExtensionEngineKind_BuiltIn)
		})
	})
}

func Test_CommandValidate(t *testing.T) {
	require := require.New(t)

	appDef := New()

	cmd := appDef.AddCommand(NewQName("test", "cmd"))

	t.Run("must error if argument name is unknown", func(t *testing.T) {
		arg := NewQName("test", "arg")
		cmd.SetArg(arg)
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, arg.String())

		_ = appDef.AddObject(arg)
	})

	t.Run("must error if unlogged argument name is unknown", func(t *testing.T) {
		unl := NewQName("test", "unl")
		cmd.SetUnloggedArg(unl)
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, unl.String())

		_ = appDef.AddObject(unl)
	})

	t.Run("must error if result object name is unknown", func(t *testing.T) {
		res := NewQName("test", "res")
		cmd.SetResult(res)
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameNotFound)
		require.ErrorContains(err, res.String())

		_ = appDef.AddObject(res)
	})

	t.Run("must error if extension name or engine is missed", func(t *testing.T) {
		_, err := appDef.Build()
		require.ErrorIs(err, ErrNameMissed)
		require.ErrorContains(err, "extension name")

		require.ErrorIs(err, ErrExtensionEngineKindMissed)
	})

	cmd.SetExtension("CmdExt", ExtensionEngineKind_BuiltIn)
	_, err := appDef.Build()
	require.NoError(err)
}
