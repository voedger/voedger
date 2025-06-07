/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package constraints_test

import (
	"math"
	"regexp"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestNewConstraint(t *testing.T) {
	type args struct {
		kind  appdef.ConstraintKind
		value any
		c     []string
	}
	tests := []struct {
		name string
		args args
		want any // should be equal type of args.value
	}{
		{"Min length",
			args{appdef.ConstraintKind_MinLen, uint16(1), []string{"test min length"}},
			1,
		},
		{"Max length",
			args{appdef.ConstraintKind_MaxLen, uint16(100), []string{"test max length"}},
			100,
		},
		{"Pattern",
			args{appdef.ConstraintKind_Pattern, "^/w+$", []string{"test pattern"}},
			regexp.MustCompile("^/w+$"),
		},
		{"Min inclusive",
			args{appdef.ConstraintKind_MinIncl, float64(1), []string{"test min inclusive"}},
			1,
		},
		{"Min exclusive",
			args{appdef.ConstraintKind_MinExcl, float64(1), []string{"test min exclusive"}},
			1,
		},
		{"Max inclusive",
			args{appdef.ConstraintKind_MaxIncl, float64(1), []string{"test max inclusive"}},
			1,
		},
		{"Max exclusive",
			args{appdef.ConstraintKind_MaxExcl, float64(1), []string{"test max exclusive"}},
			1,
		},
		{"string enumeration",
			args{appdef.ConstraintKind_Enum, []string{"c", "b", "a", "b"}, []string{"test string enum"}},
			[]string{"a", "b", "c"},
		},
		{"int32 enumeration",
			args{appdef.ConstraintKind_Enum, []int32{3, 2, 1, 3}, []string{"test int32 enum"}},
			[]int32{1, 2, 3},
		},
		{"int64 enumeration",
			args{appdef.ConstraintKind_Enum, []int64{3, 2, 1, 2}, nil},
			[]int64{1, 2, 3},
		},
		{"float32 enumeration",
			args{appdef.ConstraintKind_Enum, []float32{1, 3, 2, 1}, []string{"test", "float32", "enum"}},
			[]float32{1, 2, 3},
		},
		{"float64 enumeration",
			args{appdef.ConstraintKind_Enum, []float64{3, 1, 2, 2, 3}, []string{"test float64 enum"}},
			[]float64{1, 2, 3},
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := constraints.NewConstraint(tt.args.kind, tt.args.value, tt.args.c...)
			require.NotNil(c)
			require.Equal(tt.args.kind, c.Kind())
			require.EqualValues(tt.want, c.Value())
			require.EqualValues(tt.args.c, c.CommentLines())
		})
	}
}

func TestNewConstraintPanics(t *testing.T) {
	type args struct {
		kind  appdef.ConstraintKind
		value any
	}
	tests := []struct {
		name string
		args args
		e    error
	}{
		{"MaxLen(0)",
			args{appdef.ConstraintKind_MaxLen, uint16(0)}, appdef.ErrOutOfBoundsError,
		},
		{"Pattern(`^[error$`)",
			args{appdef.ConstraintKind_Pattern, `^[error$`}, nil,
		},
		{"MinIncl(+∞)",
			args{appdef.ConstraintKind_MinIncl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MinIncl(+∞)",
			args{appdef.ConstraintKind_MinIncl, math.Inf(+1)}, appdef.ErrOutOfBoundsError,
		},
		{"MinExcl(NaN)",
			args{appdef.ConstraintKind_MinExcl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MinExcl(+∞)",
			args{appdef.ConstraintKind_MinExcl, math.Inf(+1)}, appdef.ErrOutOfBoundsError,
		},
		{"MaxIncl(NaN)",
			args{appdef.ConstraintKind_MaxIncl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MaxIncl(-∞)",
			args{appdef.ConstraintKind_MaxIncl, math.Inf(-1)}, appdef.ErrOutOfBoundsError,
		},
		{"MaxExcl(NaN)",
			args{appdef.ConstraintKind_MaxExcl, math.NaN()}, appdef.ErrInvalidError,
		},
		{"MaxExcl(-∞)",
			args{appdef.ConstraintKind_MaxExcl, math.Inf(-1)}, appdef.ErrOutOfBoundsError,
		},
		{"Enum([]string{})",
			args{appdef.ConstraintKind_Enum, []string{}}, appdef.ErrMissedError,
		},
		{"Enum([]bool)",
			args{appdef.ConstraintKind_Enum, []bool{true, false}}, appdef.ErrUnsupportedError,
		},
		{"Enum([][]byte)",
			args{appdef.ConstraintKind_Enum, [][]byte{{1, 2, 3}, {4, 5, 6}}}, appdef.ErrUnsupportedError,
		},
		{"???(0)",
			args{appdef.ConstraintKind_count, 0}, appdef.ErrUnsupportedError,
		},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.e == nil {
				require.Panics(func() { _ = constraints.NewConstraint(tt.args.kind, tt.args.value) })
			} else {
				require.Panics(func() { _ = constraints.NewConstraint(tt.args.kind, tt.args.value) },
					require.Is(tt.e))
			}
		})
	}
}
