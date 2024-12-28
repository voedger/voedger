/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package datas_test

import (
	"fmt"
	"maps"
	"regexp"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_AppDef_AddData(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	intName := appdef.NewQName("test", "int")
	strName := appdef.NewQName("test", "string")
	tokenName := appdef.NewQName("test", "token")

	t.Run("should be ok to add data types", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddData(intName, appdef.DataKind_int64, appdef.NullQName)
		_ = wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName)
		token := wsb.AddData(tokenName, appdef.DataKind_string, strName)
		token.AddConstraints(constraints.MinLen(1), constraints.MaxLen(100), constraints.Pattern(`^\w+$`, "only word characters allowed"))

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to find builded data type", func(t *testing.T) {
			i := appdef.Data(tested.Type, intName)
			require.Equal(appdef.TypeKind_Data, i.Kind())
			require.Equal(intName, i.QName())
			require.Equal(appdef.DataKind_int64, i.DataKind())
			require.False(i.IsSystem())
			require.Equal(appdef.SysData(tested.Type, appdef.DataKind_int64), i.Ancestor())

			s := appdef.Data(tested.Type, strName)
			require.Equal(appdef.TypeKind_Data, s.Kind())
			require.Equal(strName, s.QName())
			require.Equal(appdef.DataKind_string, s.DataKind())
			require.Equal(appdef.SysData(tested.Type, appdef.DataKind_string), s.Ancestor())

			tk := appdef.Data(tested.Type, tokenName)
			require.Equal(appdef.TypeKind_Data, tk.Kind())
			require.Equal(tokenName, tk.QName())
			require.Equal(appdef.DataKind_string, tk.DataKind())
			require.Equal(s, tk.Ancestor())
			cnt := 0
			for k, c := range tk.Constraints(false) {
				cnt++
				switch k {
				case appdef.ConstraintKind_MinLen:
					require.Equal(appdef.ConstraintKind_MinLen, c.Kind())
					require.EqualValues(1, c.Value())
				case appdef.ConstraintKind_MaxLen:
					require.Equal(appdef.ConstraintKind_MaxLen, c.Kind())
					require.EqualValues(100, c.Value())
				case appdef.ConstraintKind_Pattern:
					require.Equal(appdef.ConstraintKind_Pattern, c.Kind())
					require.EqualValues(`^\w+$`, c.Value().(*regexp.Regexp).String())
					require.Equal("only word characters allowed", c.Comment())
				default:
					require.Failf("unexpected constraint", "constraint: %v", c)
				}
			}
			require.Equal(3, cnt)
		})

		t.Run("should be ok to enum data types", func(t *testing.T) {
			cnt := 0
			for d := range appdef.DataTypes(tested.Types()) {
				if !d.IsSystem() {
					cnt++
					require.Equal(appdef.TypeKind_Data, d.Kind())
					switch cnt {
					case 1:
						require.Equal(intName, d.QName())
					case 2:
						require.Equal(strName, d.QName())
					case 3:
						require.Equal(tokenName, d.QName())
					default:
						require.Failf("unexpected data type", "data type: %v", d)
					}
				}
			}
			require.Equal(3, cnt)
		})

		require.Nil(appdef.Data(tested.Type, appdef.NewQName("test", "unknown")), "check nil returns")
	}

	testWith(app)
	testWith(app.Workspace(wsName))

	t.Run("should be panics", func(t *testing.T) {

		t.Run("if invalid data type name", func(t *testing.T) {
			wsb := builder.New().AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(appdef.NewQName("naked", "🔫"), appdef.DataKind_QName, appdef.NullQName)
			}, require.Is(appdef.ErrInvalidError), require.Has("naked.🔫"))
		})

		t.Run("if type with name already exists", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			wsb.AddObject(intName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64, appdef.NullQName)
			}, require.Is(appdef.ErrAlreadyExistsError), require.Has(intName.String()))
		})

		t.Run("if sys data to inherits from not found", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(strName, appdef.DataKind_null, appdef.NullQName)
			}, require.Is(appdef.ErrNotFoundError), require.Has("null"))
		})

		t.Run("if ancestor not found", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() {
				wsb.AddData(strName, appdef.DataKind_string,
					appdef.NewQName("test", "unknown"), // <- error here
				)
			}, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
		})

		t.Run("if ancestor is not data type", func(t *testing.T) {
			objName := appdef.NewQName("test", "object")
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			_ = wsb.AddObject(objName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64,
					objName, // <- error here
				)
			}, require.Is(appdef.ErrNotFoundError), require.Has(objName.String()))
		})

		t.Run("if ancestor has different kind", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			_ = wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName)
			require.Panics(func() {
				wsb.AddData(intName, appdef.DataKind_int64, strName)
			}, require.Is(appdef.ErrInvalidError), require.Has(strName.String()))
		})

		t.Run("if incompatible constraints", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			require.Panics(func() { _ = wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName, constraints.MinIncl(1)) },
				require.Is(appdef.ErrIncompatibleError), require.Has("MinIncl"))
			require.Panics(func() { _ = wsb.AddData(intName, appdef.DataKind_float64, appdef.NullQName, constraints.MaxLen(100)) },
				require.Is(appdef.ErrIncompatibleError), require.Has("MaxLen"))
		})
	})
}

func Test_SystemDataTypes(t *testing.T) {
	require := require.New(t)

	app, err := builder.New().Build()
	require.NoError(err)

	t.Run("must be ok to get system data types", func(t *testing.T) {
		sysWS := app.Workspace(appdef.SysWorkspaceQName)
		for k := appdef.DataKind_null + 1; k < appdef.DataKind_FakeLast; k++ {
			d := appdef.SysData(app.Type, k)
			require.NotNil(d)
			require.Equal(appdef.SysDataName(k), d.QName())
			require.Equal(appdef.TypeKind_Data, d.Kind())
			require.Equal(d.Workspace(), sysWS)
			require.True(d.IsSystem())
			require.Equal(k, d.DataKind())
			require.Nil(d.Ancestor())
			require.Empty(maps.Collect(d.Constraints(false)))
		}
	})
}

func Test_NewAnonymousData(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	docName := appdef.NewQName("test", "doc")

	t.Run("should be ok to add anonymous data types", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		doc := wsb.AddODoc(docName)
		doc.AddField("str", appdef.DataKind_string, false, constraints.MinLen(1), constraints.MaxLen(100))

		t.Run("should be ok to build", func(t *testing.T) {
			a, err := adb.Build()
			require.NoError(err)
			require.NotNil(a)

			app = a
		})
	})

	require.NotNil(app)

	testWith := func(tested types.IWithTypes) {
		t.Run("should be ok to inspect anonymous data type", func(t *testing.T) {
			doc := appdef.ODoc(tested.Type, docName)
			require.NotNil(doc)

			str := doc.Field("str")
			require.NotNil(str)

			data := str.Data()
			require.NotNil(data)
			require.Equal(appdef.NullQName, data.QName())
			require.Equal(appdef.TypeKind_Data, data.Kind())
			require.Equal(appdef.DataKind_string, data.DataKind())
			require.False(data.IsSystem())
			require.Equal(appdef.SysData(tested.Type, appdef.DataKind_string), data.Ancestor())
			require.Equal(`string-data`, fmt.Sprint(data))

			cnt := 0
			for k, c := range data.Constraints(false) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(appdef.ConstraintKind_MinLen, k)
					require.EqualValues(1, c.Value())
					require.Equal(`MinLen: 1`, fmt.Sprint(c))
				case 2:
					require.Equal(appdef.ConstraintKind_MaxLen, k)
					require.EqualValues(100, c.Value())
					require.Equal(`MaxLen: 100`, fmt.Sprint(c))
				default:
					require.Failf("unexpected constraint", "constraint: %v", c)
				}
			}
			require.Equal(2, cnt)
		})
	}

	testWith(app)
	testWith(app.Workspace(wsName))
}
