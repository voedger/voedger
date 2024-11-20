/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_IsSysField(t *testing.T) {
	type args struct {
		name FieldName
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true if sys.QName",
			args: args{SystemField_QName},
			want: true,
		},
		{
			name: "true if sys.ID",
			args: args{SystemField_ID},
			want: true,
		},
		{
			name: "true if sys.ParentID",
			args: args{SystemField_ParentID},
			want: true,
		},
		{
			name: "true if sys.Container",
			args: args{SystemField_Container},
			want: true,
		},
		{
			name: "true if sys.IsActive",
			args: args{SystemField_IsActive},
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
			if got := IsSysField(tt.args.name); got != tt.want {
				t.Errorf("sysField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_AddField(t *testing.T) {
	require := require.New(t)

	wsName := NewQName("test", "workspace")
	objName := NewQName("test", "object")

	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(wsName)

	obj := wsb.AddObject(objName)
	require.NotNil(obj)

	t.Run("should be ok to add field", func(t *testing.T) {
		obj.AddField("f1", DataKind_int64, true)

		app, err := adb.Build()
		require.NoError(err)

		obj := Object(app.Type, objName)
		require.Equal([]IField{obj.Field("f1")}, obj.UserFields())
		require.Equal(1, obj.UserFieldCount())
		require.Equal(obj.UserFieldCount()+2, obj.FieldCount()) // + sys.QName + sys.Container

		f := obj.Field("f1")
		require.NotNil(f)
		require.Equal("f1", f.Name())
		require.False(f.IsSys())

		require.Equal(DataKind_int64, f.DataKind())
		require.True(f.IsFixedWidth())
		require.True(f.DataKind().IsFixed())

		require.True(f.Required())
		require.False(f.Verifiable())
	})

	t.Run("should be ok to add field use chain notation", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		_ = wsb.AddObject(objName).
			AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_int32, false).
			AddField("f3", DataKind_string, false)

		app, err := adb.Build()
		require.NoError(err)

		obj := Object(app.Type, objName)
		require.Equal(3, obj.UserFieldCount())
		require.Equal(3+2, obj.FieldCount()) // + sys.QName + sys.Container

		require.NotNil(obj.Field("f1"))
		require.NotNil(obj.Field("f2"))
		require.NotNil(obj.Field("f3"))
		require.Equal(DataKind_string, obj.Field("f3").DataKind())
	})

	t.Run("should be panics", func(t *testing.T) {
		require.Panics(func() { obj.AddField("", DataKind_int64, true) },
			require.Is(ErrMissedError))
		require.Panics(func() { obj.AddField("naked_🔫", DataKind_int64, true) },
			require.Is(ErrInvalidError),
			require.Has("naked_🔫"))
		require.Panics(func() { obj.AddField("f1", DataKind_int64, true) },
			require.Is(ErrAlreadyExistsError),
			require.Has("f1"))

		t.Run("if field data kind is not allowed by type kind", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddField("f1", DataKind_Event, false) },
				require.Is(ErrIncompatibleError),
				require.Has("Event"))
		})

		t.Run("if too many fields", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			for i := 0; i < MaxTypeFieldCount-2; i++ { // -2 because sys.QName, sys.Container
				o.AddField(fmt.Sprintf("f_%#x", i), DataKind_bool, false)
			}
			require.Panics(func() { o.AddField("errorField", DataKind_bool, true) },
				require.Is(ErrTooManyError))
		})

		t.Run("if not found field data kind", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddField("errorField", DataKind_FakeLast, false) },
				require.Is(ErrNotFoundError))
		})

		t.Run("if unknown data type", func(t *testing.T) {
			adb := New()
			adb.AddPackage("test", "test.com/test")
			wsb := adb.AddWorkspace(wsName)
			o := wsb.AddObject(objName)
			require.Panics(func() { o.AddDataField("errorField", NewQName("test", "unknown"), false) },
				require.Is(ErrNotFoundError),
				require.Has("test.unknown"))
		})
	})
}

func Test_SetFieldComment(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsName := NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	objName := NewQName("test", "object")
	wsb.AddObject(objName).
		AddField("f1", DataKind_int64, true).
		SetFieldComment("f1", "test comment")

	app, err := adb.Build()
	require.NoError(err)

	t.Run("should be ok to obtain field comment", func(t *testing.T) {
		obj := Object(app.Type, objName)
		require.Equal(1, obj.UserFieldCount())
		f1 := obj.Field("f1")
		require.NotNil(f1)
		require.Equal("test comment", f1.Comment())
	})

	t.Run("should be panic if unknown field name passed to comment", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		require.Panics(func() {
			wsb.AddObject(NewQName("test", "object")).
				SetFieldComment("unknownField", "error here")
		}, require.Is(ErrNotFoundError), require.Has("unknownField"))
	})
}

func Test_SetFieldVerify(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsName := NewQName("test", "workspace")
	wsb := adb.AddWorkspace(wsName)

	objName := NewQName("test", "object")
	wsb.AddObject(objName).
		AddField("f1", DataKind_int64, true).
		SetFieldVerify("f1", VerificationKind_Phone).
		AddField("f2", DataKind_int64, true).
		SetFieldVerify("f2", VerificationKind_Any...)

	app, err := adb.Build()
	require.NoError(err)

	t.Run("should be ok to obtain verified field", func(t *testing.T) {
		obj := Object(app.Type, objName)
		require.Equal(2, obj.UserFieldCount())
		f1 := obj.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(VerificationKind_EMail))
		require.True(f1.VerificationKind(VerificationKind_Phone))
		require.False(f1.VerificationKind(VerificationKind_FakeLast))

		f2 := obj.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(VerificationKind_EMail))
		require.True(f2.VerificationKind(VerificationKind_Phone))
		require.False(f2.VerificationKind(VerificationKind_FakeLast))
	})

	t.Run("should be panic if unknown field name passed to verify", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(wsName)
		require.Panics(func() {
			wsb.AddObject(NewQName("test", "object")).
				SetFieldVerify("unknownField", VerificationKind_Phone)
		}, require.Is(ErrNotFoundError), require.Has("unknownField"))
	})
}

func Test_AddRefField(t *testing.T) {
	require := require.New(t)

	docName := NewQName("test", "doc")
	var app IAppDef

	t.Run("should be ok to add reference fields", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(NewQName("test", "workspace"))

		doc := wsb.AddWDoc(docName)
		require.NotNil(doc)

		doc.
			AddField("f1", DataKind_int64, true).
			AddRefField("rf1", true).
			AddRefField("rf2", false, docName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to work with reference fields", func(t *testing.T) {
		doc := WDoc(app.Type, docName)

		t.Run("should be ok type cast reference field", func(t *testing.T) {
			fld := doc.Field("rf1")
			require.NotNil(fld)

			require.Equal("rf1", fld.Name())
			require.Equal(DataKind_RecordID, fld.DataKind())
			require.True(fld.Required())

			rf, ok := fld.(IRefField)
			require.True(ok)
			require.Empty(rf.Refs())
		})

		t.Run("should be ok to obtain reference field", func(t *testing.T) {
			rf2 := doc.RefField("rf2")
			require.NotNil(rf2)

			require.Equal("rf2", rf2.Name())
			require.Equal(DataKind_RecordID, rf2.DataKind())
			require.False(rf2.Required())

			require.EqualValues(QNames{docName}, rf2.Refs())
		})

		t.Run("should be nil if unknown reference field", func(t *testing.T) {
			require.Nil(doc.RefField("unknown"))
			require.Nil(doc.RefField("f1"), "must be nil because `f1` is not a reference field")
		})

		t.Run("should be ok to enumerate reference fields", func(t *testing.T) {
			require.Equal(2, func() int {
				cnt := 0
				for _, rf := range doc.RefFields() {
					cnt++
					switch cnt {
					case 1:
						require.Equal(doc.RefField("rf1"), rf)
						require.True(rf.Ref(docName))
						require.True(rf.Ref(NewQName("test", "unknown")), "must be ok because any links are allowed in the field rf1")
					case 2:
						require.EqualValues(QNames{docName}, rf.Refs())
						require.True(rf.Ref(docName))
						require.False(rf.Ref(NewQName("test", "unknown")))
					default:
						require.Failf("unexpected reference field", "field name: %s", rf.Name())
					}
				}
				return cnt
			}())
		})
	})

	t.Run("should be panic if empty field name", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")
		wsb := adb.AddWorkspace(NewQName("test", "workspace"))
		doc := wsb.AddWDoc(docName)
		require.Panics(func() { doc.AddRefField("", false) },
			require.Is(ErrMissedError))
	})
}

func Test_UserFields(t *testing.T) {
	require := require.New(t)

	docName := NewQName("test", "doc")
	var app IAppDef

	t.Run("should be ok to add fields", func(t *testing.T) {
		adb := New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(NewQName("test", "workspace"))

		doc := wsb.AddODoc(docName)
		require.NotNil(doc)

		doc.
			AddField("f", DataKind_int64, true).
			AddField("vf", DataKind_string, true).SetFieldVerify("vf", VerificationKind_EMail).
			AddRefField("rf", true, docName)

		a, err := adb.Build()
		require.NoError(err)

		app = a
	})

	t.Run("should be ok to enumerate user fields", func(t *testing.T) {
		doc := ODoc(app.Type, docName)
		require.Equal(3, doc.UserFieldCount())

		require.Equal(doc.UserFieldCount(), func() int {
			cnt := 0
			for _, f := range doc.Fields() {
				if !f.IsSys() {
					cnt++
					switch cnt {
					case 1:
						require.Equal(doc.Field("f"), f)
					case 2:
						require.True(f.VerificationKind(VerificationKind_EMail))
					case 3:
						require.EqualValues(QNames{docName}, f.(IRefField).Refs())
					default:
						require.Failf("unexpected reference field", "field name: %s", f.Name())
					}
				}
			}
			return cnt
		}())
	})
}

func TestValidateRefFields(t *testing.T) {
	require := require.New(t)

	adb := New()
	adb.AddPackage("test", "test.com/test")

	wsb := adb.AddWorkspace(NewQName("test", "workspace"))

	doc := wsb.AddCDoc(NewQName("test", "doc"))
	doc.AddRefField("f1", true, NewQName("test", "rec"))

	rec := wsb.AddCRecord(NewQName("test", "rec"))
	rec.AddRefField("f1", true, NewQName("test", "rec"))

	t.Run("should be ok if all reference field is valid", func(t *testing.T) {
		_, err := adb.Build()
		require.NoError(err)
	})

	t.Run("should be error", func(t *testing.T) {
		objName := NewQName("test", "obj")

		t.Run("if reference field refs to unknown", func(t *testing.T) {
			rec.AddRefField("f2", true, objName)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has(objName.String()))
		})

		t.Run("if reference field refs to non referable type", func(t *testing.T) {
			_ = wsb.AddObject(objName)
			_, err := adb.Build()
			require.Error(err, require.Is(ErrNotFoundError), require.Has(objName.String()))
		})
	})
}

func TestNullFields(t *testing.T) {
	require := require.New(t)

	require.Nil(NullFields.Field("field"))
	require.Zero(NullFields.FieldCount())
	require.Empty(NullFields.Fields())

	require.Nil(NullFields.RefField("field"))
	require.Empty(NullFields.RefFields())

	require.Zero(NullFields.UserFieldCount())
	require.Empty(NullFields.UserFields())
}

func TestVerificationKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{
			name: "0 —> `VerificationKind_EMail`",
			k:    VerificationKind_EMail,
			want: `VerificationKind_EMail`,
		},
		{
			name: "1 —> `VerificationKind_Phone`",
			k:    VerificationKind_Phone,
			want: `VerificationKind_Phone`,
		},
		{
			name: "3 —> `3`",
			k:    VerificationKind_FakeLast + 1,
			want: `VerificationKind(3)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.String(); got != tt.want {
				t.Errorf("VerificationKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerificationKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{
			name: `0 —> "VerificationKind_EMail"`,
			k:    VerificationKind_EMail,
			want: `"VerificationKind_EMail"`,
		},
		{
			name: `1 —> "VerificationKind_Phone"`,
			k:    VerificationKind_Phone,
			want: `"VerificationKind_Phone"`,
		},
		{
			name: "2 —> 2",
			k:    VerificationKind_FakeLast,
			want: `2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalJSON()
			if err != nil {
				t.Errorf("VerificationKind.MarshalJSON() return unexpected error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("VerificationKind.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerificationKind_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    VerificationKind
		wantErr bool
	}{
		{
			name:    `0 —> VerificationKind_Email`,
			data:    `0`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `1 —> VerificationKind_Phone`,
			data:    `1`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `2 —> VerificationKind(2)`,
			data:    `2`,
			want:    VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `3 —> VerificationKind(3)`,
			data:    `3`,
			want:    VerificationKind(3),
			wantErr: false,
		},
		{
			name:    `"VerificationKind_EMail" —> VerificationKind_EMail`,
			data:    `"VerificationKind_EMail"`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"VerificationKind_Phone" —> VerificationKind_Phone`,
			data:    `"VerificationKind_Phone"`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"0" —> VerificationKind_Email`,
			data:    `"0"`,
			want:    VerificationKind_EMail,
			wantErr: false,
		},
		{
			name:    `"1" —> VerificationKind_Phone`,
			data:    `"1"`,
			want:    VerificationKind_Phone,
			wantErr: false,
		},
		{
			name:    `"2" —> VerificationKind(2)`,
			data:    `"2"`,
			want:    VerificationKind(2),
			wantErr: false,
		},
		{
			name:    `"3" —> VerificationKind(3)`,
			data:    `"3"`,
			want:    VerificationKind(3),
			wantErr: false,
		},
		{
			name:    `65536 —> error`,
			data:    `65536`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `-1 —> error`,
			data:    `-1`,
			want:    0,
			wantErr: true,
		},
		{
			name:    `"abc" —> error`,
			data:    `"abc"`,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var k VerificationKind
			err := k.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("VerificationKind.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if k != tt.want {
					t.Errorf("VerificationKind.UnmarshalJSON(%v) result = %v, want %v", tt.data, k, tt.want)
				}
			}
		})
	}
}

func TestVerificationKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    VerificationKind
		want string
	}{
		{name: "basic test", k: VerificationKind_EMail, want: "EMail"},
		{name: "out of range", k: VerificationKind_FakeLast + 1, want: (VerificationKind_FakeLast + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(VerificationKind).TrimString() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}
}
