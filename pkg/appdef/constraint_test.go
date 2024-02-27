/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Maxim Geraskin
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
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
		{"string enumeration",
			args{ConstraintKind_Enum, []string{"c", "b", "a", "b"}, []string{"test string enum"}},
			args{ConstraintKind_Enum, []string{"a", "b", "c"}, []string{"test string enum"}},
		},
		{"int32 enumeration",
			args{ConstraintKind_Enum, []int32{3, 2, 1, 3}, []string{"test int32 enum"}},
			args{ConstraintKind_Enum, []int32{1, 2, 3}, []string{"test int32 enum"}},
		},
		{"int64 enumeration",
			args{ConstraintKind_Enum, []int64{3, 2, 1, 2}, []string{}},
			args{ConstraintKind_Enum, []int64{1, 2, 3}, []string{}},
		},
		{"float32 enumeration",
			args{ConstraintKind_Enum, []float32{1, 3, 2, 1}, []string{"test", "float32", "enum"}},
			args{ConstraintKind_Enum, []float32{1, 2, 3}, []string{"test", "float32", "enum"}},
		},
		{"float64 enumeration",
			args{ConstraintKind_Enum, []float64{3, 1, 2, 2, 3}, []string{"test float64 enum"}},
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
		{"MaxLen(0)",
			args{ConstraintKind_MaxLen, uint16(0)},
		},
		{"Pattern(`^[error$`)",
			args{ConstraintKind_Pattern, `^[error$`},
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, math.NaN()},
		},
		{"MinIncl(+∞)",
			args{ConstraintKind_MinIncl, math.Inf(+1)},
		},
		{"MinExcl(NaN)",
			args{ConstraintKind_MinExcl, math.NaN()},
		},
		{"MinExcl(+∞)",
			args{ConstraintKind_MinExcl, math.Inf(+1)},
		},
		{"MaxIncl(NaN)",
			args{ConstraintKind_MaxIncl, math.NaN()},
		},
		{"MaxIncl(-∞)",
			args{ConstraintKind_MaxIncl, math.Inf(-1)},
		},
		{"MaxExcl(NaN)",
			args{ConstraintKind_MaxExcl, math.NaN()},
		},
		{"MaxExcl(-∞)",
			args{ConstraintKind_MaxExcl, math.Inf(-1)},
		},
		{"Enum([]string{})",
			args{ConstraintKind_Enum, []string{}},
		},
		{"Enum([]bool)",
			args{ConstraintKind_Enum, []bool{true, false}},
		},
		{"Enum([][]byte)",
			args{ConstraintKind_Enum, [][]byte{{1, 2, 3}, {4, 5, 6}}},
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

func Test_dataConstraint_String(t *testing.T) {
	tests := []struct {
		name  string
		c     IConstraint
		wantS string
	}{
		{"MinLen", MinLen(1), "MinLen: 1"},
		{"MaxLen", MaxLen(100), "MaxLen: 100"},
		{"Pattern", Pattern(`^\d+$`), "Pattern: `^\\d+$`"},
		{"MinIncl", MinIncl(1), "MinIncl: 1"},
		{"MinExcl", MinExcl(0), "MinExcl: 0"},
		{"MinExcl(-∞)", MinExcl(math.Inf(-1)), "MinExcl: -Inf"},
		{"MaxIncl", MaxIncl(100), "MaxIncl: 100"},
		{"MaxExcl", MaxExcl(100), "MaxExcl: 100"},
		{"MaxExcl(+∞)", MaxExcl(math.Inf(+1)), "MaxExcl: +Inf"},
		{"Enum(string)", Enum("c", "d", "a", "a", "b", "c"), "Enum: [a b c d]"},
		{"Enum(float64)", Enum(float64(1), 2, 3, 4, math.Round(100*math.Pi)/100, math.Inf(-1)), "Enum: [-Inf 1 2 3 3.14 4]"},
		{"Enum(long case)", Enum("b", "d", "a", strings.Repeat("c", 100)), "Enum: [a b cccccccccccccccccccccccccccccccccccccccccccccccccccc…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := fmt.Sprint(tt.c); gotS != tt.wantS {
				t.Errorf("dataConstraint.String() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func TestConstraintKind_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		k    ConstraintKind
		want string
	}{
		{
			name: `0 —> "ConstraintKind_null"`,
			k:    ConstraintKind_null,
			want: `ConstraintKind_null`,
		},
		{
			name: `1 —> "ConstraintKind_MinLen"`,
			k:    ConstraintKind_MinLen,
			want: `ConstraintKind_MinLen`,
		},
		{
			name: `ConstraintKind_Count —> 4`,
			k:    ConstraintKind_Count,
			want: strconv.FormatUint(uint64(ConstraintKind_Count), 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.k.MarshalText()
			if err != nil {
				t.Errorf("%T.MarshalText() unexpected error %v", tt.k, err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("%T.MarshalText() = %v, want %v", tt.k, got, tt.want)
			}
		})
	}

	t.Run("100% cover", func(t *testing.T) {
		const tested = ConstraintKind_Count + 1
		want := "ConstraintKind(" + strconv.FormatInt(int64(tested), 10) + ")"
		got := tested.String()
		if got != want {
			t.Errorf("(ConstraintKind_Count + 1).String() = %v, want %v", got, want)
		}
	})
}

func TestConstraintKind_TrimString(t *testing.T) {
	tests := []struct {
		name string
		k    ConstraintKind
		want string
	}{
		{name: "basic", k: ConstraintKind_MinLen, want: "MinLen"},
		{name: "out of range", k: ConstraintKind_Count + 1, want: (ConstraintKind_Count + 1).String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.TrimString(); got != tt.want {
				t.Errorf("%v.(%T).TrimString() = %v, want %v", tt.k, tt.k, got, tt.want)
			}
		})
	}
}
