/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddCommand(t *testing.T) {
	require := require.New(t)

	var app IAppDef
	cmdName, parName, unlName, resName := NewQName("test", "cmd"), NewQName("test", "par"), NewQName("test", "unl"), NewQName("test", "res")

	t.Run("must be ok to add command", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		_ = adb.AddObject(parName)
		_ = adb.AddObject(unlName)
		_ = adb.AddObject(resName)

		cmd := adb.AddCommand(cmdName)

		t.Run("must be ok to assign cmd parameter and result", func(t *testing.T) {
			cmd.SetEngine(ExtensionEngineKind_BuiltIn)
			cmd.
				SetParam(parName).
				SetResult(resName)
			cmd.SetUnloggedParam(unlName)
		})

		t.Run("must be ok to build", func(t *testing.T) {
			a, err := adb.Build()
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
		require.Equal(cmdName.Entity(), cmd.Name())
		require.Equal(c, cmd)

		require.Equal(ExtensionEngineKind_BuiltIn, cmd.Engine())

		require.Equal(parName, cmd.Param().QName())
		require.Equal(TypeKind_Object, cmd.Param().Kind())

		require.Equal(unlName, cmd.UnloggedParam().QName())
		require.Equal(TypeKind_Object, cmd.UnloggedParam().Kind())

		require.Equal(resName, cmd.Result().QName())
		require.Equal(TypeKind_Object, cmd.Result().Kind())
	})

	t.Run("must be ok to enum commands", func(t *testing.T) {
		cnt := 0
		for c := range app.Commands {
			cnt++
			switch cnt {
			case 1:
				require.Equal(cmdName, c.QName())
			default:
				require.Failf("unexpected command", "command: %v", c)
			}
		}
		require.Equal(1, cnt)
	})

	t.Run("check nil returns", func(t *testing.T) {
		unknown := NewQName("test", "unknown")
		require.Nil(app.Command(unknown))
	})

	t.Run("panic if name is empty", func(t *testing.T) {
		adb := New()
		require.Panics(func() { adb.AddCommand(NullQName) },
			require.Is(ErrMissedError))
	})

	t.Run("panic if name is invalid", func(t *testing.T) {
		adb := New()
		require.Panics(func() { adb.AddCommand(NewQName("naked", "🔫")) },
			require.Is(ErrInvalidError),
			require.Has("naked.🔫"))
	})

	t.Run("panic if type with name already exists", func(t *testing.T) {
		testName := NewQName("test", "dupe")
		adb := New()
		adb.AddPackage("test", "test.com/test")
		adb.AddObject(testName)
		require.Panics(func() { adb.AddCommand(testName) },
			require.Is(ErrAlreadyExistsError),
			require.Has(testName.String()))
	})

	t.Run("panic if extension name is empty", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		cmd := adb.AddCommand(NewQName("test", "cmd"))
		require.Panics(func() { cmd.SetName("") },
			require.Is(ErrMissedError),
			require.Has("test.cmd"))
	})

	t.Run("panic if extension name is invalid", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		cmd := adb.AddCommand(NewQName("test", "cmd"))
		require.Panics(func() { cmd.SetName("naked 🔫") },
			require.Is(ErrInvalidError),
			require.Has("naked 🔫"))
	})

	t.Run("panic if extension kind is invalid", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		cmd := adb.AddCommand(NewQName("test", "cmd"))
		require.Panics(func() { cmd.SetEngine(ExtensionEngineKind_null) },
			require.Is(ErrOutOfBoundsError))
		require.Panics(func() { cmd.SetEngine(ExtensionEngineKind_Count) },
			require.Is(ErrOutOfBoundsError))
	})
}

func Test_CommandValidate(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")
	obj := NewQName("test", "obj")
	_ = adb.AddObject(obj)
	bad := NewQName("test", "workspace")
	_ = adb.AddWorkspace(bad)
	unknown := NewQName("test", "unknown")

	cmd := adb.AddCommand(NewQName("test", "cmd"))

	t.Run("errors in parameter", func(t *testing.T) {
		t.Run("must error if parameter name is unknown", func(t *testing.T) {
			cmd.SetParam(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has(unknown))
		})

		t.Run("must error if deprecated parameter type", func(t *testing.T) {
			cmd.SetParam(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrInvalidError), require.Has(bad))
		})

		cmd.SetParam(obj)
	})

	t.Run("errors in unlogged parameter", func(t *testing.T) {
		t.Run("must error if unlogged parameter name is unknown", func(t *testing.T) {
			cmd.SetUnloggedParam(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has(unknown))
		})

		t.Run("must error if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetUnloggedParam(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrInvalidError), require.Has(bad))
		})

		cmd.SetUnloggedParam(obj)
	})

	t.Run("errors in result", func(t *testing.T) {
		t.Run("must error if result object name is unknown", func(t *testing.T) {
			cmd.SetResult(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has(unknown))
		})

		t.Run("must error if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetResult(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrInvalidError), require.Has(bad))
		})

		cmd.SetResult(obj)
	})

	_, err := adb.Build()
	require.NoError(err)
}
