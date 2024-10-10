/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func Test_AppDefExtensions(t *testing.T) {

	require := require.New(t)

	var app IAppDef

	cmdName := NewQName("test", "cmd")
	qrName := NewQName("test", "query")
	prjName := NewQName("test", "projector")
	parName := NewQName("test", "param")
	resName := NewQName("test", "res")

	sysViews := NewQName("sys", "views")
	viewName := NewQName("test", "view")

	t.Run("Should be ok to build application with extensions", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		cmd := adb.AddCommand(cmdName)
		cmd.SetEngine(ExtensionEngineKind_WASM)
		cmd.
			SetParam(parName).
			SetResult(resName)

		qry := adb.AddQuery(qrName)
		qry.
			SetParam(parName).
			SetResult(QNameANY)

		prj := adb.AddProjector(prjName)
		prj.Events().
			Add(cmdName, ProjectorEventKind_Execute)
		prj.Intents().
			Add(sysViews, viewName)

		_ = adb.AddObject(parName)
		_ = adb.AddObject(resName)

		v := adb.AddView(viewName)
		v.Key().PartKey().AddField("pk", DataKind_int64)
		v.Key().ClustCols().AddField("cc", DataKind_string)
		v.Value().AddField("f1", DataKind_int64, true)

		a, err := adb.Build()
		require.NoError(err)

		app = a
		require.NotNil(app)
	})

	t.Run("Should be ok to enumerate extensions", func(t *testing.T) {
		var extNames []QName
		for ex := range app.Extensions {
			extNames = append(extNames, ex.QName())
		}
		require.Len(extNames, 3)
		require.Equal([]QName{cmdName, prjName, qrName}, extNames)
	})

	t.Run("Should be ok to find extension by name", func(t *testing.T) {
		ext := app.Extension(cmdName)
		require.NotNil(ext)
		require.Equal(cmdName, ext.QName())

		require.Nil(app.Extension(NewQName("test", "unknown")), "Should be nil if unknown extension")
	})
}

func TestExtensionEngineKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ExtensionEngineKind
		want string
	}{
		{name: `0 —> "ExtensionEngineKind_null"`,
			k:    ExtensionEngineKind_null,
			want: `ExtensionEngineKind_null`,
		},
		{name: `1 —> "ExtensionEngineKind_BuiltIn"`,
			k:    ExtensionEngineKind_BuiltIn,
			want: `ExtensionEngineKind_BuiltIn`,
		},
		{name: `ExtensionEngineKind_Count —> <number>`,
			k:    ExtensionEngineKind_Count,
			want: utils.UintToString(ExtensionEngineKind_Count),
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
		const tested = ExtensionEngineKind_Count + 1
		want := "ExtensionEngineKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ExtensionEngineKind_Count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestExtensionEngineKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    ExtensionEngineKind
		want string
	}{
		{name: "basic", k: ExtensionEngineKind_BuiltIn, want: "BuiltIn"},
		{name: "out of range", k: ExtensionEngineKind_Count + 1, want: (ExtensionEngineKind_Count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(ExtensionEngineKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
