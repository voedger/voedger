/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func Test_AppDefExtensions(t *testing.T) {

	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")

	cmdName := appdef.NewQName("test", "cmd")
	qrName := appdef.NewQName("test", "query")
	prjName := appdef.NewQName("test", "projector")
	parName := appdef.NewQName("test", "param")
	resName := appdef.NewQName("test", "res")

	sysViews := appdef.NewQName("sys", "views")
	viewName := appdef.NewQName("test", "view")

	t.Run("Should be ok to build application with extensions", func(t *testing.T) {
		adb := appdef.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(parName)
		_ = wsb.AddObject(resName)

		cmd := wsb.AddCommand(cmdName)
		cmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := wsb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(appdef.QNameANY)

		prj := wsb.AddProjector(prjName, []appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(cmdName))
		prj.Intents().
			Add(sysViews, viewName)

		v := wsb.AddView(viewName)
		v.Key().PartKey().AddField("pk", appdef.DataKind_int64)
		v.Key().ClustCols().AddField("cc", appdef.DataKind_string)
		v.Value().AddField("f1", appdef.DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
		require.NotNil(app)
	})

	testWith := func(tested testedTypes) {
		t.Run("should be ok to enumerate extensions", func(t *testing.T) {
			var extNames []appdef.QName
			for ex := range appdef.Extensions(tested.Types()) {
				require.Equal(wsName, ex.Workspace().QName())
				extNames = append(extNames, ex.QName())
			}
			require.Len(extNames, 3)
			require.Equal([]appdef.QName{cmdName, prjName, qrName}, extNames)
		})

		t.Run("should be ok to find extension by name", func(t *testing.T) {
			ext := appdef.Extension(tested.Type, cmdName)
			require.NotNil(ext)
			require.Equal(cmdName, ext.QName())
		})

		require.Nil(appdef.Extension(tested.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown")
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}

func TestExtensionEngineKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.ExtensionEngineKind
		want string
	}{
		{name: `0 —> "ExtensionEngineKind_null"`,
			k:    appdef.ExtensionEngineKind_null,
			want: `ExtensionEngineKind_null`,
		},
		{name: `1 —> "ExtensionEngineKind_BuiltIn"`,
			k:    appdef.ExtensionEngineKind_BuiltIn,
			want: `ExtensionEngineKind_BuiltIn`,
		},
		{name: `ExtensionEngineKind_count —> <number>`,
			k:    appdef.ExtensionEngineKind_count,
			want: utils.UintToString(appdef.ExtensionEngineKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("ExtensionEngineKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("ExtensionEngineKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover ExtensionEngineKind.String()", func(t *testing.T) {
		const tested = appdef.ExtensionEngineKind_count + 1
		want := "ExtensionEngineKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ExtensionEngineKind_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestExtensionEngineKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.ExtensionEngineKind
		want string
	}{
		{name: "basic", k: appdef.ExtensionEngineKind_BuiltIn, want: "BuiltIn"},
		{name: "out of range", k: appdef.ExtensionEngineKind_count + 1, want: (appdef.ExtensionEngineKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ExtensionEngineKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
