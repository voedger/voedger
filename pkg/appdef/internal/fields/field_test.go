/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package fields_test

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/internal/fields"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_IsSysField(t *testing.T) {
	type args struct {
		name appdef.FieldName
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.QName",
			args: args{appdef.SystemField_QName},
			want: true,
		},
		{
			name: "true if sys.ID",
			args: args{appdef.SystemField_ID},
			want: true,
		},
		{
			name: "true if sys.ParentID",
			args: args{appdef.SystemField_ParentID},
			want: true,
		},
		{
			name: "true if sys.Container",
			args: args{appdef.SystemField_Container},
			want: true,
		},
		{
			name: "true if sys.IsActive",
			args: args{appdef.SystemField_IsActive},
			want: true,
		},
		{
			name: "false if empty",
			args: args{""},
			want: false,
		},
		{
			name: "false if basic user",
			args: args{"userField"},
			want: false,
		},
		{
			name: "false if curious user",
			args: args{"sys.user"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appdef.IsSysField(tt.args.name); got != tt.want {
				t.Errorf("sysField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Fields(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	objName := appdef.NewQName("test", "object")

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(wsName)

	var app appdef.IAppDef

	t.Run("should be ok to add fields", func(t *testing.T) {
		obj := wsb.AddObject(objName)
		require.NotNil(obj)
		obj.AddField("f1", appdef.DataKind_int64, true,
			constraints.MinIncl(0), constraints.MaxIncl(100)).
			AddDataField("f2", appdef.SysData_String, false, constraints.Enum("male", "female"))

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to inspect fields", func(t *testing.T) {
		obj := appdef.Object(app.Type, objName)

		cnt := 0
		t.Run("should be ok to enum fields", func(t *testing.T) {
			for _, f := range obj.Fields() {
				switch cnt {
				case 0:
					require.Equal(appdef.SystemField_QName, f.Name())
				case 1:
					require.Equal(appdef.SystemField_Container, f.Name())
				case 2:
					require.Equal("f1", f.Name())
				case 3:
					require.Equal("f2", f.Name())
				default:
					require.Fail("unexpected field", "field: %v", f)
				}
				cnt++
			}
			require.Equal(4, cnt)
		})

		t.Run("should be ok to inspect user fields", func(t *testing.T) {
			require.Equal([]appdef.IField{obj.Field("f1"), obj.Field("f2")}, obj.UserFields())
			require.Equal(2, obj.UserFieldCount())
			require.Equal(obj.UserFieldCount()+2, obj.FieldCount()) // + sys.QName + sys.Container
		})

		t.Run("should be ok to inspect field", func(t *testing.T) {
			f := obj.Field("f1")
			require.NotNil(f)
			require.Equal("f1", f.Name())
			require.False(f.IsSys())

			require.Equal(appdef.DataKind_int64, f.DataKind())
			require.True(f.IsFixedWidth())
			require.True(f.DataKind().IsFixed())

			require.True(f.Required())
			require.False(f.Verifiable())

			cc := f.Constraints()
			require.Len(cc, 2)
			require.Contains(cc, appdef.ConstraintKind_MinIncl)
			require.Contains(cc, appdef.ConstraintKind_MaxIncl)
			require.EqualValues(0, cc[appdef.ConstraintKind_MinIncl].Value())
			require.EqualValues(100, cc[appdef.ConstraintKind_MaxIncl].Value())

			require.Equal(`int64-field «f1»`, fmt.Sprint(f))
		})
	})
}

func Test_FieldsPanics(t *testing.T) {
	require := require.New(t)

	wsName := appdef.NewQName("test", "workspace")
	objName := appdef.NewQName("test", "object")

	t.Run("should be panics", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		obj := wsb.AddObject(objName)

		require.Panics(func() { obj.AddField("", appdef.DataKind_int64, true) },
			require.Is(appdef.ErrMissedError))
		require.Panics(func() { obj.AddField("naked_🔫", appdef.DataKind_int64, true) },
			require.Is(appdef.ErrInvalidError),
			require.Has("naked_🔫"))
		obj.AddField("f1", appdef.DataKind_int64, true)
		require.Panics(func() { obj.AddField("f1", appdef.DataKind_int64, true) },
			require.Is(appdef.ErrAlreadyExistsError),
			require.Has("f1"))

		t.Run("if field data kind is not allowed by type kind", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddField("f1", appdef.DataKind_Event, false) },
				require.Is(appdef.ErrIncompatibleError),
				require.Has("Event"))
		})

		t.Run("if too many fields", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			for i := 0; i < appdef.MaxTypeFieldCount-2; i++ { // -2 because sys.QName, sys.Container
				o.AddField(fmt.Sprintf("f_%#x", i), appdef.DataKind_bool, false)
			}
			require.Panics(func() { o.AddField("errorField", appdef.DataKind_bool, true) },
				require.Is(appdef.ErrTooManyError))
		})

		t.Run("if not found field data kind", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddField("errorField", appdef.DataKind_FakeLast, false) },
				require.Is(appdef.ErrNotFoundError))
		})

		t.Run("if unknown data type", func(t *testing.T) {
			adb := builder.New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddDataField("errorField", appdef.NewQName("test", "unknown"), false) },
				require.Is(appdef.ErrNotFoundError),
				require.Has("test.unknown"))
		})
	})
}

func Test_SetFieldComment(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	objName := appdef.NewQName("test", "object")
	wsb.AddObject(objName).
		AddField("f1", appdef.DataKind_int64, true).
		SetFieldComment("f1", "field comment").
		AddRefField("f2", true).
		SetFieldComment("f2", "ref comment")

	app, err := adb.Build()
	require.NoError(err)

	t.Run("should be ok to obtain field comment", func(t *testing.T) {
		obj := appdef.Object(app.Type, objName)
		require.Equal("field comment", obj.Field("f1").Comment())
		require.Equal("ref comment", obj.Field("f2").Comment())
	})

	t.Run("should be panic if unknown field name passed to comment", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		require.Panics(func() {
			wsb.AddObject(appdef.NewQName("test", "object")).
				SetFieldComment("unknownField", "error here")
		}, require.Is(appdef.ErrNotFoundError), require.Has("unknownField"))
	})
}

func Test_SetFieldVerify(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsName := appdef.NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	objName := appdef.NewQName("test", "object")
	wsb.AddObject(objName).
		AddField("f1", appdef.DataKind_int64, true).
		SetFieldVerify("f1", appdef.VerificationKind_Phone).
		AddField("f2", appdef.DataKind_int64, true).
		SetFieldVerify("f2", appdef.VerificationKind_Any...)

	app, err := adb.Build()
	require.NoError(err)

	t.Run("should be ok to obtain verified field", func(t *testing.T) {
		obj := appdef.Object(app.Type, objName)
		require.Equal(2, obj.UserFieldCount())
		f1 := obj.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(appdef.VerificationKind_EMail))
		require.True(f1.VerificationKind(appdef.VerificationKind_Phone))
		require.False(f1.VerificationKind(appdef.VerificationKind_FakeLast))

		f2 := obj.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(appdef.VerificationKind_EMail))
		require.True(f2.VerificationKind(appdef.VerificationKind_Phone))
		require.False(f2.VerificationKind(appdef.VerificationKind_FakeLast))
	})

	t.Run("should be panic if unknown field name passed to verify", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		require.Panics(func() {
			wsb.AddObject(appdef.NewQName("test", "object")).
				SetFieldVerify("unknownField", appdef.VerificationKind_Phone)
		}, require.Is(appdef.ErrNotFoundError), require.Has("unknownField"))
	})
}

func Test_AddRefField(t *testing.T) {
	require := require.New(t)

	docName := appdef.NewQName("test", "doc")
	var app appdef.IAppDef

	t.Run("should be ok to add reference fields", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

		doc := wsb.AddWDoc(docName)
		require.NotNil(doc)

		doc.
			AddField("f1", appdef.DataKind_int64, true).
			AddRefField("rf1", true).
			AddRefField("rf2", false, docName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to work with reference fields", func(t *testing.T) {
		doc := appdef.WDoc(app.Type, docName)

		t.Run("should be ok type cast reference field", func(t *testing.T) {
			fld := doc.Field("rf1")
			require.NotNil(fld)

			require.Equal("rf1", fld.Name())
			require.Equal(appdef.DataKind_RecordID, fld.DataKind())
			require.True(fld.Required())

			rf, ok := fld.(appdef.IRefField)
			require.True(ok)
			require.Empty(rf.Refs())
		})

		t.Run("should be ok to obtain reference field", func(t *testing.T) {
			rf2 := doc.RefField("rf2")
			require.NotNil(rf2)

			require.Equal("rf2", rf2.Name())
			require.Equal(appdef.DataKind_RecordID, rf2.DataKind())
			require.False(rf2.Required())

			require.EqualValues([]appdef.QName{docName}, rf2.Refs())
		})

		t.Run("should be nil if unknown reference field", func(t *testing.T) {
			require.Nil(doc.RefField("unknown"))
			require.Nil(doc.RefField("f1"), "must be nil because `f1` is not a reference field")
		})

		t.Run("should be ok to enumerate reference fields", func(t *testing.T) {
			require.Equal(2, func() int {
				cnt := 0
				for _, rf := range doc.RefFields() {
					if rf.IsSys() {
						continue
					}
					cnt++
					switch cnt {
					case 1:
						require.Equal(doc.RefField("rf1"), rf)
						require.True(rf.Ref(docName))
						require.True(rf.Ref(appdef.NewQName("test", "unknown")), "must be ok because any links are allowed in the field rf1")
					case 2:
						require.EqualValues([]appdef.QName{docName}, rf.Refs())
						require.True(rf.Ref(docName))
						require.False(rf.Ref(appdef.NewQName("test", "unknown")))
					default:
						require.Failf("unexpected reference field", "field name: %s", rf.Name())
					}
				}
				return cnt
			}())
		})
	})

	t.Run("should be panic if empty field name", func(t *testing.T) {
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))
		doc := wsb.AddWDoc(docName)
		require.Panics(func() { doc.AddRefField("", false) },
			require.Is(appdef.ErrMissedError))
	})
}

func TestValidateRefFields(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	doc := wsb.AddCDoc(appdef.NewQName("test", "doc"))
	doc.AddRefField("f1", true, appdef.NewQName("test", "rec"))

	rec := wsb.AddCRecord(appdef.NewQName("test", "rec"))
	rec.AddRefField("f1", true, appdef.NewQName("test", "rec"))

	t.Run("should be ok if all reference field is valid", func(t *testing.T) {
		_, err := adb.Build()
		require.NoError(err)
	})

	t.Run("should be error", func(t *testing.T) {
		objName := appdef.NewQName("test", "obj")

		t.Run("if reference field refs to unknown", func(t *testing.T) {
			rec.AddRefField("f2", true, objName)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(objName.String()))
		})

		t.Run("if reference field refs to non referable type", func(t *testing.T) {
			_ = wsb.AddObject(objName)
			_, err := adb.Build()
			require.Error(err, require.Is(appdef.ErrNotFoundError), require.Has(objName.String()))
		})
	})
}

func TestExportedRoutines(t *testing.T) {
	require := require.New(t)

	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

	ff := fields.MakeWithFields(wsb.Workspace(), appdef.TypeKind_Object)

	t.Run("should be ok to use AddDataField", func(t *testing.T) {
		fields.AddDataField(&ff, "f1", appdef.SysData_bool, false)
		require.Equal("f1", ff.Field("f1").Name())
	})

	t.Run("should be ok to use AddField", func(t *testing.T) {
		fields.AddField(&ff, "f2", appdef.DataKind_bool, false)
		require.Equal("f2", ff.Field("f2").Name())
	})

	t.Run("should be ok to use AddRefField", func(t *testing.T) {
		fields.AddRefField(&ff, "f3", false)
		require.Equal("f3", ff.Field("f3").Name())
	})

	t.Run("should be ok to use SetFieldComment", func(t *testing.T) {
		fields.SetFieldComment(&ff, "f1", "test")
		require.Equal("test", ff.Field("f1").Comment())
	})

	t.Run("should be ok to use SetFieldVerify", func(t *testing.T) {
		fields.SetFieldVerify(&ff, "f2", appdef.VerificationKind_EMail)
		require.True(ff.Field("f2").VerificationKind(appdef.VerificationKind_EMail))
	})
}
