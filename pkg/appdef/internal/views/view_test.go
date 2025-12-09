/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package views_test

import (
	"regexp"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/internal/types"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Views(t *testing.T) {
	require := require.New(t)

	var app appdef.IAppDef

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	numName := appdef.NewQName("test", "natural")
	_ = wsb.AddData(numName, appdef.DataKind_int64, appdef.NullQName, constraints.MinExcl(0))

	digsData := appdef.NewQName("test", "digs")
	_ = wsb.AddData(digsData, appdef.DataKind_string, appdef.NullQName, constraints.Pattern(`^\d+$`, "only digits allowed"))

	docName := appdef.NewQName("test", "doc")
	_ = wsb.AddCDoc(docName)

	kbName := appdef.NewQName("test", "KB")
	_ = wsb.AddData(kbName, appdef.DataKind_bytes, appdef.NullQName, constraints.MinLen(1), constraints.MaxLen(1024, "up to 1 KB"))

	viewName := appdef.NewQName("test", "view")
	vb := wsb.AddView(viewName)

	t.Run("should be ok to build view", func(t *testing.T) {

		vb.SetComment("test view")

		t.Run("should be ok to add partition key fields", func(t *testing.T) {
			vb.Key().PartKey().AddDataField("pkF1", numName)
			vb.Key().PartKey().AddField("pkF2", appdef.DataKind_bool)

			t.Run("panic if field already exists in view", func(t *testing.T) {
				require.Panics(func() {
					vb.Key().PartKey().AddField("pkF1", appdef.DataKind_int64)
				}, require.Is(appdef.ErrAlreadyExistsError), require.Has("pkF1"))
			})

			t.Run("panic if variable length field added to pk", func(t *testing.T) {
				require.Panics(func() {
					vb.Key().PartKey().AddField("pkF3", appdef.DataKind_string)
				}, require.Is(appdef.ErrUnsupportedError), require.Has("pkF3"))
				require.Panics(func() {
					vb.Key().PartKey().AddDataField("pkF3", digsData)
				}, require.Is(appdef.ErrUnsupportedError), require.Has("pkF3"))
			})

			t.Run("panic if unknown data type field added to pk", func(t *testing.T) {
				require.Panics(func() {
					vb.Key().PartKey().AddDataField("pkF3", appdef.NewQName("test", "unknown"))
				}, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
			})
		})

		t.Run("should be ok to add clustering columns fields", func(t *testing.T) {
			vb.Key().ClustCols().AddField("ccF1", appdef.DataKind_int64)
			vb.Key().ClustCols().AddRefField("ccF2", docName)

			t.Run("panic if field already exists in view", func(t *testing.T) {
				require.Panics(func() {
					vb.Key().ClustCols().AddField("ccF1", appdef.DataKind_int64)
				}, require.Is(appdef.ErrAlreadyExistsError), require.Has("ccF1"))
			})

			t.Run("panic if unknown data type field added to cc", func(t *testing.T) {
				require.Panics(func() {
					vb.Key().ClustCols().AddDataField("ccF3", appdef.NewQName("test", "unknown"))
				}, require.Is(appdef.ErrNotFoundError), require.Has("test.unknown"))
			})
		})

		t.Run("should be ok to add value fields", func(t *testing.T) {
			vb.Value().
				AddField("valF1", appdef.DataKind_bool, true).
				AddDataField("valF2", digsData, false, constraints.MaxLen(100)).SetFieldComment("valF2", "up to 100 digits")
		})

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith_1 := func(tested types.IWithTypes) {
		view := appdef.View(tested.Type, viewName)

		t.Run("should be ok to read view", func(t *testing.T) {
			require.Equal("test view", view.Comment())
			require.Equal(viewName, view.QName())
			require.Equal(appdef.TypeKind_ViewRecord, view.Kind())

			checkValueValF2 := func(f appdef.IField) {
				require.Equal("valF2", f.Name())
				require.False(f.Required())
				cnt := 0
				for k, c := range f.Constraints() {
					cnt++
					switch k {
					case appdef.ConstraintKind_MaxLen:
						require.EqualValues(100, c.Value())
					case appdef.ConstraintKind_Pattern:
						require.EqualValues(`^\d+$`, c.Value().(*regexp.Regexp).String())
						require.Equal("only digits allowed", c.Comment())
					default:
						require.Fail("unexpected constraint", "constraint: %v", c)
					}
				}
				require.Equal("up to 100 digits", f.Comment())
			}

			require.Equal(7, view.FieldCount())
			cnt := 0
			for _, f := range view.Fields() {
				cnt++
				switch cnt {
				case 1:
					require.Equal(appdef.SystemField_QName, f.Name())
					require.True(f.IsSys())
				case 2:
					require.Equal("pkF1", f.Name())
					require.Equal(numName, f.Data().QName())
					require.True(f.Required())
				case 3:
					require.Equal("pkF2", f.Name())
					require.True(f.Required())
				case 4:
					require.Equal("ccF1", f.Name())
					require.False(f.Required())
				case 5:
					require.Equal("ccF2", f.Name())
					require.False(f.Required())
				case 6:
					require.Equal("valF1", f.Name())
					require.True(f.Required())
				case 7:
					checkValueValF2(f)
				default:
					require.Fail("unexpected field", f.Name())
				}
			}
			require.Equal(view.FieldCount(), cnt)

			t.Run("should be ok to read view full key", func(t *testing.T) {
				key := view.Key()
				require.Equal(4, key.FieldCount())
				cnt := 0
				for _, f := range key.Fields() {
					cnt++
					switch cnt {
					case 1:
						require.Equal("pkF1", f.Name())
						require.Equal(numName, f.Data().QName())
						require.True(f.Required())
					case 2:
						require.Equal("pkF2", f.Name())
						require.True(f.Required())
					case 3:
						require.Equal("ccF1", f.Name())
						require.False(f.Required())
					case 4:
						require.Equal("ccF2", f.Name())
						require.False(f.Required())
					default:
						require.Fail("unexpected field", f.Name())
					}
				}
				require.Equal(key.FieldCount(), cnt)
			})

			t.Run("should be ok to read view partition key", func(t *testing.T) {
				pk := view.Key().PartKey()
				require.Equal(2, pk.FieldCount())
				cnt := 0
				for _, f := range pk.Fields() {
					cnt++
					switch cnt {
					case 1:
						require.Equal("pkF1", f.Name())
						require.Equal(numName, f.Data().QName())
						require.True(f.Required())
					case 2:
						require.Equal("pkF2", f.Name())
						require.True(f.Required())
					default:
						require.Fail("unexpected field", f.Name())
					}
				}
				require.Equal(pk.FieldCount(), cnt)
			})

			t.Run("should be ok to read view clustering columns", func(t *testing.T) {
				cc := view.Key().ClustCols()
				require.Equal(2, cc.FieldCount())
				cnt := 0
				for _, f := range cc.Fields() {
					cnt++
					switch cnt {
					case 1:
						require.Equal("ccF1", f.Name())
						require.False(f.Required())
					case 2:
						require.Equal("ccF2", f.Name())
						require.False(f.Required())
					default:
						require.Fail("unexpected field", f.Name())
					}
				}
				require.Equal(cc.FieldCount(), cnt)
			})

			t.Run("should be ok to read view value", func(t *testing.T) {
				val := view.Value()
				require.Equal(3, val.FieldCount())
				cnt := 0
				for _, f := range val.Fields() {
					cnt++
					switch cnt {
					case 1:
						require.Equal(appdef.SystemField_QName, f.Name())
						require.True(f.IsSys())
					case 2:
						require.Equal("valF1", f.Name())
						require.True(f.Required())
					case 3:
						checkValueValF2(f)
					default:
						require.Fail("unexpected field", f.Name())
					}
				}
				require.Equal(val.FieldCount(), cnt)
			})

			t.Run("should be ok to cast Type() as appdef.IView", func(t *testing.T) {
				typ := tested.Type(viewName)
				require.NotNil(typ)
				require.Equal(appdef.TypeKind_ViewRecord, typ.Kind())

				v, ok := typ.(appdef.IView)
				require.True(ok)
				require.Equal(v, view)
			})

			require.Nil(appdef.View(tested.Type, appdef.NewQName("test", "unknown")), "should be nil if unknown view")

			t.Run("should be nil if not view", func(t *testing.T) {
				require.Nil(appdef.View(tested.Type, docName))

				typ := tested.Type(docName)
				require.NotNil(typ)
				v, ok := typ.(appdef.IView)
				require.False(ok)
				require.Nil(v)
			})
		})

		t.Run("should be ok to enum views", func(t *testing.T) {
			names := appdef.QNames{}
			for v := range appdef.Views(tested.Types()) {
				if !v.IsSystem() {
					names.Add(v.QName())
				}
			}
			require.Len(names, 1)
			require.Contains(names, viewName)
		})
	}

	testWith_1(app)
	testWith_1(app.Workspace(wsName))

	t.Run("should be ok to add fields to view after app build", func(t *testing.T) {
		vb.Key().PartKey().
			AddRefField("pkF3", docName).
			SetFieldComment("pkF3", "test comment")

		vb.Key().ClustCols().
			AddDataField("ccF3", kbName).SetFieldComment("ccF3", "one KB")

		t.Run("panic if add second variable length field", func(t *testing.T) {
			require.Panics(func() {
				vb.Key().ClustCols().AddField("ccF3_1", appdef.DataKind_bytes)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("ccF3"))
			require.Panics(func() {
				vb.Key().ClustCols().AddDataField("ccF3_1", kbName)
			}, require.Is(appdef.ErrUnsupportedError), require.Has("ccF3"))
		})

		vb.Value().
			AddRefField("valF3", false, docName).
			AddField("valF4", appdef.DataKind_bytes, false, constraints.MaxLen(1024)).SetFieldComment("valF4", "test comment").
			AddField("valF5", appdef.DataKind_bool, false).SetFieldVerify("valF5", appdef.VerificationKind_EMail)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	testWith_2 := func(tested types.IWithTypes) {
		view := appdef.View(tested.Type, viewName)

		require.Equal(3, view.Key().PartKey().FieldCount())
		require.Equal(3, view.Key().ClustCols().FieldCount())
		require.Equal(6, view.Key().FieldCount())

		require.Equal(view.Key().FieldCount()+view.Value().FieldCount(), view.FieldCount())

		require.Equal("test comment", view.Key().Field("pkF3").Comment())
		require.Equal("one KB", view.Key().Field("ccF3").Comment())

		require.Equal(5, view.Value().UserFieldCount())

		cnt := 0
		for _, f := range view.Value().Fields() {
			cnt++
			switch f.Name() {
			case appdef.SystemField_QName:
				require.Equal(appdef.DataKind_QName, f.DataKind())
				require.True(f.IsSys())
			case "valF1":
				require.Equal(appdef.DataKind_bool, f.DataKind())
				require.True(f.Required())
			case "valF2":
				require.Equal(appdef.DataKind_string, f.DataKind())
				require.False(f.Required())
				// valF2 constraints checked above
			case "valF3":
				require.Equal(appdef.DataKind_RecordID, f.DataKind())
				require.False(f.Required())
				require.EqualValues([]appdef.QName{docName}, f.(appdef.IRefField).Refs())
			case "valF4":
				require.Equal(appdef.DataKind_bytes, f.DataKind())
				require.False(f.Required())
				require.Equal("test comment", f.Comment())
				cnt := 0
				for _, c := range f.Constraints() {
					cnt++
					switch cnt {
					case 1:
						require.Equal(appdef.ConstraintKind_MaxLen, c.Kind())
						require.EqualValues(1024, c.Value())
					default:
						require.Fail("unexpected constraint", "constraint: %v", c)
					}
				}
				require.EqualValues(1, cnt)
			case "valF5":
				require.Equal(appdef.DataKind_bool, f.DataKind())
				require.False(f.Required())
				require.True(f.Verifiable())
				require.True(f.VerificationKind(appdef.VerificationKind_EMail))
				require.False(f.VerificationKind(appdef.VerificationKind_Phone))
			default:
				require.Fail("unexpected value field", "field name: %s", f.Name())
			}
		}
		require.Equal(view.Value().UserFieldCount()+1, cnt)
	}

	testWith_2(app)
	testWith_2(app.Workspace(wsName))
}

func TestViewValidate(t *testing.T) {
	require := require.New(t)

	viewName := appdef.NewQName("test", "view")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	v := wsb.AddView(viewName)
	require.NotNil(v)

	t.Run("should be error if no pkey fields", func(t *testing.T) {
		_, err := adb.Build()
		require.ErrorIs(err, appdef.ErrMissedError)
	})

	v.Key().PartKey().AddField("pk1", appdef.DataKind_bool)

	t.Run("should be error if no ccols fields", func(t *testing.T) {
		_, err := adb.Build()
		require.ErrorIs(err, appdef.ErrMissedError)
	})

	v.Key().ClustCols().AddField("cc1", appdef.DataKind_bool)

	v.Value().AddRefField("vf1", false, appdef.NewQName("test", "unknown"))
	t.Run("should be error if errors in fields", func(t *testing.T) {
		_, err := adb.Build()
		require.Error(err, require.Is(appdef.ErrNotFoundError), require.HasAll("vf1", "test.unknown"))
	})
}
