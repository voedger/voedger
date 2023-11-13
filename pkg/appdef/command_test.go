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
	cmdName, parName, unlName, resName := NewQName("test", "cmd"), NewQName("test", "par"), NewQName("test", "unl"), NewQName("test", "res")

	t.Run("must be ok to add command", func(t *testing.T) {
		appDef := New()

		_ = appDef.AddObject(parName)
		_ = appDef.AddObject(unlName)
		_ = appDef.AddObject(resName)

		cmd := appDef.AddCommand(cmdName)
		require.Equal(TypeKind_Command, cmd.Kind())
		require.Equal(cmd, appDef.Command(cmdName))
		require.Nil(cmd.Param())
		require.Nil(cmd.UnloggedParam())
		require.Nil(cmd.Result())

		t.Run("must be ok to assign cmd parameter and result", func(t *testing.T) {
			cmd.
				SetParam(parName).(ICommandBuilder).SetUnloggedParam(unlName).
				SetResult(resName).
				SetExtension("CmdExt", ExtensionEngineKind_BuiltIn, "comment")
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

		require.Equal(parName, cmd.Param().QName())
		require.Equal(TypeKind_Object, cmd.Param().Kind())

		require.Equal(unlName, cmd.UnloggedParam().QName())
		require.Equal(TypeKind_Object, cmd.UnloggedParam().Kind())

		require.Equal(resName, cmd.Result().QName())
		require.Equal(TypeKind_Object, cmd.Result().Kind())

		require.Equal("CmdExt", cmd.Extension().Name())
		require.Equal(ExtensionEngineKind_BuiltIn, cmd.Extension().Engine())
		require.Equal("comment", cmd.Extension().Comment())
	})

	t.Run("must be ok to enum functions", func(t *testing.T) {
		cnt := 0
		app.Functions(func(f IFunction) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(TypeKind_Command, f.Kind())
				require.Equal(cmdName, f.QName())
			default:
				require.Failf("unexpected function", "kind: %v, name: %v", f.Kind(), f.QName())
			}
		})
		require.Equal(1, cnt)
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
	obj := NewQName("test", "obj")
	_ = appDef.AddObject(obj)
	bad := NewQName("test", "workspace")
	_ = appDef.AddWorkspace(bad)
	unknown := NewQName("test", "unknown")

	cmd := appDef.AddCommand(NewQName("test", "cmd"))

	t.Run("errors in parameter", func(t *testing.T) {
		t.Run("must error if parameter name is unknown", func(t *testing.T) {
			cmd.SetParam(unknown)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, unknown.String())
		})

		t.Run("must error if deprecated parameter type", func(t *testing.T) {
			cmd.SetParam(bad)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrInvalidTypeKind)
			require.ErrorContains(err, bad.String())
		})

		cmd.SetParam(obj)
	})

	t.Run("errors in unlogged parameter", func(t *testing.T) {
		t.Run("must error if unlogged parameter name is unknown", func(t *testing.T) {
			cmd.SetUnloggedParam(unknown)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, unknown.String())
		})

		t.Run("must error if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetUnloggedParam(bad)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrInvalidTypeKind)
			require.ErrorContains(err, bad.String())
		})

		cmd.SetUnloggedParam(obj)
	})

	t.Run("errors in result", func(t *testing.T) {
		t.Run("must error if result object name is unknown", func(t *testing.T) {
			cmd.SetResult(unknown)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrNameNotFound)
			require.ErrorContains(err, unknown.String())
		})

		t.Run("must error if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetResult(bad)
			_, err := appDef.Build()
			require.ErrorIs(err, ErrInvalidTypeKind)
			require.ErrorContains(err, bad.String())
		})

		cmd.SetResult(obj)
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
