/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_def_AddField(t *testing.T) {
	require := require.New(t)

	def := New().AddObject(NewQName("test", "object"))
	require.NotNil(def)

	t.Run("must be ok to add field", func(t *testing.T) {
		def.AddField("f1", DataKind_int64, true)

		require.Equal(1, def.UserFieldCount())
		require.Equal(def.UserFieldCount()+1, def.FieldCount()) // + sys.QName

		f := def.Field("f1")
		require.NotNil(f)
		require.Equal("f1", f.Name())
		require.False(f.IsSys())

		require.Equal(DataKind_int64, f.DataKind())
		require.True(f.IsFixedWidth())
		require.True(f.DataKind().IsFixed())

		require.True(f.Required())
		require.False(f.Verifiable())
	})

	t.Run("chain notation is ok to add fields", func(t *testing.T) {
		d := New().AddObject(NewQName("test", "obj"))
		n := d.AddField("f1", DataKind_int64, true).
			AddField("f2", DataKind_int32, false).
			AddField("f3", DataKind_string, false).(IDef).QName()
		require.Equal(d.QName(), n)
		require.Equal(3, d.UserFieldCount())
	})

	t.Run("must be panic if empty field name", func(t *testing.T) {
		require.Panics(func() { def.AddVerifiedField("", DataKind_int64, true, VerificationKind_Phone) })
	})

	t.Run("must be panic if invalid field name", func(t *testing.T) {
		require.Panics(func() { def.AddField("naked_🔫", DataKind_int64, true) })
	})

	t.Run("must be panic if field name dupe", func(t *testing.T) {
		require.Panics(func() { def.AddField("f1", DataKind_int64, true) })
	})

	t.Run("must be panic if field data kind is not allowed by definition kind", func(t *testing.T) {
		view := New().AddView(NewQName("test", "view"))
		require.Panics(func() { view.AddPartField("f1", DataKind_string) })
	})

	t.Run("must be panic if too many fields", func(t *testing.T) {
		d := New().AddObject(NewQName("test", "obj"))
		for i := 0; i < MaxDefFieldCount-1; i++ { // -1 for sys.QName field
			d.AddField(fmt.Sprintf("f_%#x", i), DataKind_bool, false)
		}
		require.Panics(func() { d.AddField("errorField", DataKind_bool, true) })
	})
}

func Test_def_AddVerifiedField(t *testing.T) {
	require := require.New(t)

	def := New().AddObject(NewQName("test", "object"))
	require.NotNil(def)

	t.Run("must be ok to add verified field", func(t *testing.T) {
		def.AddVerifiedField("f1", DataKind_int64, true, VerificationKind_Phone)
		def.AddVerifiedField("f2", DataKind_int64, true, VerificationKind_Any...)

		require.Equal(3, def.FieldCount()) // + sys.QName
		f1 := def.Field("f1")
		require.NotNil(f1)

		require.True(f1.Verifiable())
		require.False(f1.VerificationKind(VerificationKind_EMail))
		require.True(f1.VerificationKind(VerificationKind_Phone))
		require.False(f1.VerificationKind(VerificationKind_FakeLast))

		f2 := def.Field("f2")
		require.NotNil(f2)

		require.True(f2.Verifiable())
		require.True(f2.VerificationKind(VerificationKind_EMail))
		require.True(f2.VerificationKind(VerificationKind_Phone))
		require.False(f2.VerificationKind(VerificationKind_FakeLast))
	})

	t.Run("must be panic if no verification kinds", func(t *testing.T) {
		require.Panics(func() { def.AddVerifiedField("f3", DataKind_int64, true) })
	})
}

func Test_IsSysField(t *testing.T) {
	type args struct {
		name string
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
