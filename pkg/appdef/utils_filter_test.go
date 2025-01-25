/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_FilterMatches(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "ws")
	cmdName := appdef.NewQName("test", "cmd")
	queryName := appdef.NewQName("test", "query")

	t.Run("should be ok to build application", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddCommand(cmdName)
		_ = wsb.AddQuery(queryName)

		var err error
		app, err = adb.Build()
		require.NoError(err)
		require.NotNil(app)
	})

	t.Run("should be ok to filter matches", func(t *testing.T) {
		filtered := appdef.FilterMatches(filter.AllWSFunctions(wsName), app.Types())
		require.Len(filtered, 2)
		for i, t := range filtered {
			switch i {
			case 0:
				require.Equal(cmdName, t.QName())
			case 1:
				require.Equal(queryName, t.QName())
			default:
				require.Fail("unexpected type", "type: %v", t)
			}
		}

		require.Equal(cmdName, appdef.FirstFilterMatch(filter.AllWSFunctions(wsName), app.Types()).QName())
		require.Nil(appdef.FirstFilterMatch(filter.QNames(appdef.NewQName("test", "unknown")), app.Types()))

		t.Run("filter matches iter should be breakable", func(t *testing.T) {
			cnt := 0
			for range filtered {
				cnt++
				break
			}
			require.Equal(1, cnt)
		})
	})
}

func TestFilterKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.FilterKind
		want string
	}{
		{name: `0 —> "FilterKind_null"`,
			k:    appdef.FilterKind_null,
			want: `FilterKind_null`,
		},
		{name: `1 —> "FilterKind_QNames"`,
			k:    appdef.FilterKind_QNames,
			want: `FilterKind_QNames`,
		},
		{name: `FilterKind_count —> <number>`,
			k:    appdef.FilterKind_count,
			want: utils.UintToString(appdef.FilterKind_count),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("FilterKind.MarshalText() unexpected error %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("FilterKind.MarshalText() = %s, want %v", got, tt.want)
			}
		})
	}

	t.Run("100% cover FilterKind.String()", func(t *testing.T) {
		const tested = appdef.FilterKind_count + 1
		want := "FilterKind(" + utils.UintToString(tested) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(FilterKind_count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestFilterKindTrimString(t *testing.T) {
	tests := []struct {
		name string
		k    appdef.FilterKind
		want string
	}{
		{name: "basic", k: appdef.FilterKind_QNames, want: "QNames"},
		{name: "out of range", k: appdef.FilterKind_count + 1, want: (appdef.FilterKind_count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(FilterKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
