/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddView(t *testing.T) {
	require := require.New(t)

	ab := New()

	docName := NewQName("test", "doc")
	_ = ab.AddCDoc(docName)

	viewName := NewQName("test", "view")
	vb := ab.AddView(viewName)

	t.Run("must be ok to build view", func(t *testing.T) {

		vb.SetComment("test view")

		t.Run("must be ok to add partition key fields", func(t *testing.T) {
			vb.KeyBuilder().PartKeyBuilder().AddField("pkF1", DataKind_int64)
			vb.KeyBuilder().PartKeyBuilder().AddField("pkF2", DataKind_bool)

			t.Run("panic if variable length field added to pk", func(t *testing.T) {
				require.Panics(func() {
					vb.KeyBuilder().PartKeyBuilder().AddField("pkF3", DataKind_string)
				})
			})
		})

		t.Run("must be ok to add clustering columns fields", func(t *testing.T) {
			vb.KeyBuilder().ClustColsBuilder().AddField("ccF1", DataKind_int64)
			vb.KeyBuilder().ClustColsBuilder().AddRefField("ccF2", docName)

			t.Run("panic if field already exists in pk", func(t *testing.T) {
				require.Panics(func() {
					vb.KeyBuilder().ClustColsBuilder().AddField("pkF1", DataKind_int64)
				})
			})
		})

		t.Run("must be ok to add value fields", func(t *testing.T) {
			vb.ValueBuilder().AddField("valF1", DataKind_bool, true)
			vb.ValueBuilder().AddStringField("valF2", false, Pattern(`^\d+$`, "only digits allowed"))
		})
	})

	app, err := ab.Build()
	require.NoError(err)
	view := app.View(viewName)

	t.Run("must be ok to read view", func(t *testing.T) {
		require.Equal("test view", view.Comment())
		require.Equal(viewName, view.QName())
		require.Equal(TypeKind_ViewRecord, view.Kind())

		require.Equal(7, view.FieldCount())
		cnt := 0
		view.Fields(func(f IField) {
			cnt++
			switch cnt {
			case 1:
				require.Equal(SystemField_QName, f.Name())
				require.True(f.IsSys())
			case 2:
				require.Equal("pkF1", f.Name())
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
				require.Equal("valF2", f.Name())
				require.False(f.Required())
			default:
				require.Fail("unexpected field «%s»", f.Name())
			}
		})
		require.Equal(view.FieldCount(), cnt)

		t.Run("must be ok to read view full key", func(t *testing.T) {
			key := view.Key()
			require.Equal(4, key.FieldCount())
			cnt := 0
			key.Fields(func(f IField) {
				cnt++
				switch cnt {
				case 1:
					require.Equal("pkF1", f.Name())
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
					require.Fail("unexpected field «%s»", f.Name())
				}
			})
			require.Equal(key.FieldCount(), cnt)
		})

		t.Run("must be ok to read view partition key", func(t *testing.T) {
			pk := view.Key().PartKey()
			require.Equal(2, pk.FieldCount())
			cnt := 0
			pk.Fields(func(f IField) {
				cnt++
				switch cnt {
				case 1:
					require.Equal("pkF1", f.Name())
					require.True(f.Required())
				case 2:
					require.Equal("pkF2", f.Name())
					require.True(f.Required())
				default:
					require.Fail("unexpected field «%s»", f.Name())
				}
			})
			require.Equal(pk.FieldCount(), cnt)
		})

		t.Run("must be ok to read view clustering columns", func(t *testing.T) {
			cc := view.Key().ClustCols()
			require.Equal(2, cc.FieldCount())
			cnt := 0
			cc.Fields(func(f IField) {
				cnt++
				switch cnt {
				case 1:
					require.Equal("ccF1", f.Name())
					require.False(f.Required())
				case 2:
					require.Equal("ccF2", f.Name())
					require.False(f.Required())
				default:
					require.Fail("unexpected field «%s»", f.Name())
				}
			})
			require.Equal(cc.FieldCount(), cnt)
		})

		t.Run("must be ok to read view value", func(t *testing.T) {
			val := view.Value()
			require.Equal(3, val.FieldCount())
			cnt := 0
			val.Fields(func(f IField) {
				cnt++
				switch cnt {
				case 1:
					require.Equal(SystemField_QName, f.Name())
					require.True(f.IsSys())
				case 2:
					require.Equal("valF1", f.Name())
					require.True(f.Required())
				case 3:
					require.Equal("valF2", f.Name())
					require.False(f.Required())
				default:
					require.Fail("unexpected field «%s»", f.Name())
				}
			})
			require.Equal(val.FieldCount(), cnt)
		})

		t.Run("must be ok to cast Type() as IView", func(t *testing.T) {
			typ := app.Type(viewName)
			require.NotNil(typ)
			require.Equal(TypeKind_ViewRecord, typ.Kind())

			v, ok := typ.(IView)
			require.True(ok)
			require.Equal(v, view)
		})

		require.Nil(ab.View(NewQName("test", "unknown")), "find unknown view must return nil")

		t.Run("must be nil if not view", func(t *testing.T) {
			require.Nil(app.View(docName))

			typ := app.Type(docName)
			require.NotNil(typ)
			v, ok := typ.(IView)
			require.False(ok)
			require.Nil(v)
		})
	})

	t.Run("must be ok to add fields to view after app build", func(t *testing.T) {
		vb.KeyBuilder().PartKeyBuilder().
			AddRefField("pkF3", docName).
			SetFieldComment("pkF3", "test comment")

		vb.KeyBuilder().ClustColsBuilder().
			AddBytesField("ccF3", 100).
			SetFieldComment("ccF3", "test comment")

		t.Run("panic if add second variable length field", func(t *testing.T) {
			require.Panics(func() {
				vb.KeyBuilder().ClustColsBuilder().AddBytesField("ccF3_1", 100)
			})
		})

		vb.ValueBuilder().
			AddRefField("valF3", false, docName).
			AddBytesField("valF4", false, MaxLen(1024)).SetFieldComment("valF4", "test comment").
			AddField("valF5", DataKind_bool, false).SetFieldVerify("valF5", VerificationKind_EMail)

		_, err := ab.Build()
		require.NoError(err)

		require.Equal(3, view.Key().PartKey().FieldCount())
		require.Equal(3, view.Key().ClustCols().FieldCount())
		require.Equal(6, view.Key().FieldCount())

		require.Equal(view.Key().FieldCount()+view.Value().FieldCount(), view.FieldCount())

		require.Equal("test comment", view.Key().Field("pkF3").Comment())
		require.Equal("test comment", view.Key().Field("ccF3").Comment())

		require.Equal(5, view.Value().UserFieldCount())

		view.Value().Fields(func(f IField) {
			switch f.Name() {
			case SystemField_QName:
				require.Equal(DataKind_QName, f.DataKind())
				require.True(f.IsSys())
			case "valF1":
				require.Equal(DataKind_bool, f.DataKind())
				require.True(f.Required())
			case "valF2":
				require.Equal(DataKind_string, f.DataKind())
				require.False(f.Required())
				cnt := 0
				f.Constraints(func(c IConstraint) {
					cnt++
					switch cnt {
					case 1:
						require.Equal(ConstraintKind_MaxLen, c.Kind())
						require.EqualValues(DefaultFieldMaxLength, c.Value())
					case 2:
						require.Equal(ConstraintKind_Pattern, c.Kind())
						require.EqualValues(`^\d+$`, c.Value().(*regexp.Regexp).String())
						require.Equal("only digits allowed", c.Comment())
					default:
						require.Fail("unexpected constraint", "constraint: %v", c)
					}
				})
				require.EqualValues(2, cnt)
			case "valF3":
				require.Equal(DataKind_RecordID, f.DataKind())
				require.False(f.Required())
				require.Equal([]QName{docName}, f.(IRefField).Refs())
			case "valF4":
				require.Equal(DataKind_bytes, f.DataKind())
				require.False(f.Required())
				require.Equal("test comment", f.Comment())
				cnt := 0
				f.Constraints(func(c IConstraint) {
					cnt++
					switch cnt {
					case 1:
						require.Equal(ConstraintKind_MaxLen, c.Kind())
						require.EqualValues(1024, c.Value())
					default:
						require.Fail("unexpected constraint", "constraint: %v", c)
					}
				})
				require.EqualValues(1, cnt)
			case "valF5":
				require.Equal(DataKind_bool, f.DataKind())
				require.False(f.Required())
				require.True(f.Verifiable())
				require.True(f.VerificationKind(VerificationKind_EMail))
				require.False(f.VerificationKind(VerificationKind_Phone))
			default:
				require.Fail("unexpected value field", "field name: %s", f.Name())
			}
		})
	})
}

func TestViewValidate(t *testing.T) {
	require := require.New(t)

	app := New()
	viewName := NewQName("test", "view")
	v := app.AddView(viewName)
	require.NotNil(v)

	t.Run("must be error if no pkey fields", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrFieldsMissed)
	})

	v.KeyBuilder().PartKeyBuilder().AddField("pk1", DataKind_bool)

	t.Run("must be error if no ccols fields", func(t *testing.T) {
		_, err := app.Build()
		require.ErrorIs(err, ErrFieldsMissed)
	})

	v.KeyBuilder().ClustColsBuilder().AddStringField("cc1", 100)
	_, err := app.Build()
	require.NoError(err)
}
