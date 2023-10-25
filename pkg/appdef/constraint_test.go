/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"math"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConstraint(t *testing.T) {
	type args struct {
		kind  ConstraintKind
		value any
		c     []string
	}
	tests := []struct {
		name string
		args args
		want args
	}{
		{"Min length",
			args{ConstraintKind_MinLen, uint16(1), []string{"test min length"}},
			args{ConstraintKind_MinLen, 1, []string{"test min length"}},
		},
		{"Max length",
			args{ConstraintKind_MaxLen, uint16(100), []string{"test max length"}},
			args{ConstraintKind_MaxLen, 100, []string{"test max length"}},
		},
		{"Pattern",
			args{ConstraintKind_Pattern, "^/w+$", []string{"test pattern"}},
			args{ConstraintKind_Pattern, regexp.MustCompile("^/w+$"), []string{"test pattern"}},
		},
		{"Min inclusive",
			args{ConstraintKind_MinIncl, float64(1), []string{"test min inclusive"}},
			args{ConstraintKind_MinIncl, 1, []string{"test min inclusive"}},
		},
		{"Min exclusive",
			args{ConstraintKind_MinExcl, float64(1), []string{"test min exclusive"}},
			args{ConstraintKind_MinExcl, 1, []string{"test min exclusive"}},
		},
		{"Max inclusive",
			args{ConstraintKind_MaxIncl, float64(1), []string{"test max inclusive"}},
			args{ConstraintKind_MaxIncl, 1, []string{"test max inclusive"}},
		},
		{"Max exclusive",
			args{ConstraintKind_MaxExcl, float64(1), []string{"test max exclusive"}},
			args{ConstraintKind_MaxExcl, 1, []string{"test max exclusive"}},
		},
		{"String enumeration",
			args{ConstraintKind_Enum, []string{"a", "b", "c"}, []string{"test string enum"}},
			args{ConstraintKind_Enum, []string{"a", "b", "c"}, []string{"test string enum"}},
		},
		{"Bytes enumeration",
			args{ConstraintKind_Enum, [][]byte{{1, 2}, {3, 4}, {5, 6}}, []string{"test bytes enum"}},
			args{ConstraintKind_Enum, [][]byte{{1, 2}, {3, 4}, {5, 6}}, []string{"test bytes enum"}},
		},
		{"int32 enumeration",
			args{ConstraintKind_Enum, []int32{1, 2, 3}, []string{"test int32 enum"}},
			args{ConstraintKind_Enum, []int32{1, 2, 3}, []string{"test int32 enum"}},
		},
		{"int64 enumeration",
			args{ConstraintKind_Enum, []int64{1, 2, 3}, []string{}},
			args{ConstraintKind_Enum, []int64{1, 2, 3}, []string{}},
		},
		{"float32 enumeration",
			args{ConstraintKind_Enum, []float32{1, 2, 3}, []string{"test", "float32", "enum"}},
			args{ConstraintKind_Enum, []float32{1, 2, 3}, []string{"test", "float32", "enum"}},
		},
		{"float64 enumeration",
			args{ConstraintKind_Enum, []float64{1, 2, 3}, []string{"test float64 enum"}},
			args{ConstraintKind_Enum, []float64{1, 2, 3}, []string{"test float64 enum"}},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConstraint(tt.args.kind, tt.args.value, tt.args.c...)
			require.NotNil(c)
			require.Equal(tt.want.kind, c.Kind())
			require.EqualValues(tt.want.value, c.Value())
			require.EqualValues(tt.want.c, c.CommentLines())
		})
	}
}

func TestNewConstraintPanics(t *testing.T) {
	type args struct {
		kind  ConstraintKind
		value any
	}
	tests := []struct {
		name string
		args args
	}{
		{"MinLen(MaxFieldLength+1)",
			args{ConstraintKind_MinLen, uint16(MaxFieldLength + 1)},
		},
		{"MaxLen(0)",
			args{ConstraintKind_MaxLen, uint16(0)},
		},
		{"MaxLen(MaxFieldLength+1)",
			args{ConstraintKind_MaxLen, uint16(MaxFieldLength + 1)},
		},
		{"Pattern(`^[error$`)",
			args{ConstraintKind_Pattern, `^[error$`},
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, float64(math.NaN())},
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, float64(math.Inf(+1))},
		},
		{"MinExcl(NaN)",
			args{ConstraintKind_MinExcl, float64(math.NaN())},
		},
		{"MinExcl(+∞)",
			args{ConstraintKind_MinExcl, float64(math.Inf(+1))},
		},
		{"MaxIncl(NaN)",
			args{ConstraintKind_MaxIncl, float64(math.NaN())},
		},
		{"MaxIncl(-∞)",
			args{ConstraintKind_MaxIncl, float64(math.Inf(-1))},
		},
		{"MaxExcl(NaN)",
			args{ConstraintKind_MaxExcl, float64(math.NaN())},
		},
		{"MaxExcl(-∞)",
			args{ConstraintKind_MaxExcl, float64(math.Inf(-1))},
		},
		{"Enum([]bool)",
			args{ConstraintKind_Enum, []bool{true, false}},
		},
		{"???(0)",
			args{ConstraintKind_Count, 0},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Panics(func() { _ = NewConstraint(tt.args.kind, tt.args.value) })
		})
	}
}
