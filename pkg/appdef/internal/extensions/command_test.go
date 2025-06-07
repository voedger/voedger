/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package extensions_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Commands(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef
	wsName := appdef.NewQName("test", "workspace")
	cmdName, parName, unlName, resName := appdef.NewQName("test", "cmd"), appdef.NewQName("test", "par"), appdef.NewQName("test", "unl"), appdef.NewQName("test", "res")

	t.Run("should be ok to add command", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(unlName)
		_ = wsb.AddObject(resName)

		cmd := wsb.AddCommand(cmdName)

		t.Run("should be ok to assign cmd parameter and result", func(t *testing.T) {
			cmd.SetEngine(appdef.ExtensionEngineKind_BuiltIn)
			cmd.
				SetParam(parName).
				SetResult(resName)
			cmd.SetUnloggedParam(unlName)
		})

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	t.Run("should be ok to find builded command", func(t *testing.T) {
		typ := app.Type(cmdName)
		require.Equal(appdef.TypeKind_Command, typ.Kind())

		c, ok := typ.(appdef.ICommand)
		require.True(ok)
		require.Equal(appdef.TypeKind_Command, c.Kind())

		cmd := appdef.Command(app.Type, cmdName)
		require.Equal(appdef.TypeKind_Command, cmd.Kind())
		require.Equal(cmdName.Entity(), cmd.Name())
		require.Equal(c, cmd)

		require.Equal(wsName, cmd.Workspace().QName())

		require.Equal(appdef.ExtensionEngineKind_BuiltIn, cmd.Engine())

		require.Equal(parName, cmd.Param().QName())
		require.Equal(appdef.TypeKind_Object, cmd.Param().Kind())

		require.Equal(unlName, cmd.UnloggedParam().QName())
		require.Equal(appdef.TypeKind_Object, cmd.UnloggedParam().Kind())

		require.Equal(resName, cmd.Result().QName())
		require.Equal(appdef.TypeKind_Object, cmd.Result().Kind())
	})

	t.Run("should be ok to enum commands", func(t *testing.T) {
		cnt := 0
		for c := range appdef.Commands(app.Types()) {
			if c.IsSystem() {
				continue
			}
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
		unknown := appdef.NewQName("test", "unknown")
		require.Nil(appdef.Command(app.Type, unknown))
	})
}

func Test_CommandValidate(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")
	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
	obj := appdef.NewQName("test", "obj")
	_ = wsb.AddObject(obj)
	bad := appdef.NewQName("test", "job")
	wsb.AddJob(bad).SetCronSchedule("@hourly")
	unknown := appdef.NewQName("test", "unknown")

	cmd := wsb.AddCommand(appdef.NewQName("test", "cmd"))

	t.Run("should be errors", func(t *testing.T) {
		t.Run("if parameter name is unknown", func(t *testing.T) {
			cmd.SetParam(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(unknown))
		})

		t.Run("if deprecated parameter type", func(t *testing.T) {
			cmd.SetParam(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrInvalidError), require.Has(bad))
		})

		cmd.SetParam(obj)
		t.Run("if unlogged parameter name is unknown", func(t *testing.T) {
			cmd.SetUnloggedParam(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(unknown))
		})

		t.Run("if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetUnloggedParam(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrInvalidError), require.Has(bad))
		})

		cmd.SetUnloggedParam(obj)
	})

	t.Run("should be errors in result", func(t *testing.T) {
		t.Run("if result object name is unknown", func(t *testing.T) {
			cmd.SetResult(unknown)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(unknown))
		})

		t.Run("if deprecated unlogged parameter type", func(t *testing.T) {
			cmd.SetResult(bad)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrInvalidError), require.Has(bad))
		})

		cmd.SetResult(obj)
	})

	_, err := adb.Build()
	require.NoError(err)
}
