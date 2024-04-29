/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package set

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Month uint8

const (
	Month_jan Month = iota
	Month_feb
	Month_mar
	Month_apr
	Month_may
	Month_jun
	Month_jul
	Month_aug
	Month_sep
	Month_oct
	Month_nov
	Month_dec

	Month_count
)

var TypeKindStr = map[Month]string{
	Month_jan: "Month_jan",
	Month_feb: "Month_feb",
	Month_mar: "Month_mar",
	Month_apr: "Month_apr",
	Month_may: "Month_may",
	Month_jun: "Month_jun",
	Month_jul: "Month_jul",
	Month_aug: "Month_aug",
	Month_sep: "Month_sep",
	Month_oct: "Month_oct",
	Month_nov: "Month_nov",
	Month_dec: "Month_dec",
}

func (t Month) String() string {
	if s, ok := TypeKindStr[t]; ok {
		return s
	}
	return fmt.Sprintf("Month(%d)", t)
}

func (t Month) TrimString() string {
	return strings.TrimPrefix(t.String(), "Month_")
}

func TestEmpty(t *testing.T) {
	require := require.New(t)
	require.Zero(Empty[Month]().Len())
}

func TestFrom(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		want string
	}{
		{"empty", From[Month](), "[]"},
		{"one", From(Month_feb), "[feb]"},
		{"two", From(Month_feb, Month_mar), "[feb mar]"},
		{"three", From(Month_feb, Month_mar, Month_oct), "[feb mar oct]"},
		{"should shrink duplicates", From(Month_aug, Month_aug), "[aug]"},
		{"should accept out of bounds", From(Month_count + 1), fmt.Sprintf("[%v]", Month_count+1)},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, tt.set.String(), "SetFrom(%v).String() = %v, want %v", tt.set, tt.set.String(), tt.want)
		})
	}
}

func TestSet_AsArray(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		want []Month
	}{
		{"empty", From[Month](), nil},
		{"one", From(Month_may), []Month{Month_may}},
		{"two", From(Month_may, Month_jun), []Month{Month_may, Month_jun}},
		{"out of bounds", From(Month_may, Month_count+1), []Month{Month_may, Month_count + 1}},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.AsArray()
			require.EqualValues(got, tt.want, "SetFrom(%v).AsArray() = %v, want %v", tt.set, got, tt.want)
		})
	}
}

func TestSet_AsInt64(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		want uint64
	}{
		{"empty", From[Month](), 0},
		{"one", From(Month_may), 1 << Month_may},
		{"two", From(Month_may, Month_jun), 1<<Month_may | 1<<Month_jun},
		{"out of bounds", From(Month_may, Month_count+1), 1<<Month_may | 1<<(Month_count+1)},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.AsInt64()
			require.EqualValues(got, tt.want, "SetFrom(%v).AsInt64() = %v, want %v", tt.set, got, tt.want)
		})
	}
}

func TestSet_Clear(t *testing.T) {
	require := require.New(t)

	t.Run("should be ok to clear one value", func(t *testing.T) {
		set := From(Month_may, Month_jun)
		set.Clear(Month_may)
		require.Equal("[jun]", set.String())
		require.Equal(1, set.Len())
		require.EqualValues([]Month{Month_jun}, set.AsArray())
	})

	t.Run("should be ok to clear a few values", func(t *testing.T) {
		set := From(Month_may, Month_jun, Month_aug)
		set.Clear(Month_may, Month_jun)
		require.Equal("[aug]", set.String())
	})

	t.Run("should be safe to clear already cleared values", func(t *testing.T) {
		set := Set[Month]{}
		set.Clear(Month_may, Month_jun)
		require.Equal("[]", set.String())
	})
}

func TestSet_ClearAll(t *testing.T) {
	set := From(Month_may, Month_jun)
	set.ClearAll()
	require.Equal(t, "[]", set.String())
	require.Zero(t, set.Len())
	require.Empty(t, set.AsArray())
}

func TestSet_Clone(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
	}{
		{"empty", Set[Month]{}},
		{"one", From(Month_may)},
		{"two", From(Month_may, Month_jun)},
	}
	require := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.set.Clone()
			require.Equal(tt.set.String(), clone.String())
			require.Equal(tt.set.Len(), clone.Len())
			require.Equal(tt.set.AsArray(), clone.AsArray())

			clone.Set(Month_dec)

			require.NotEqual(tt.set.String(), clone.String())
			require.Equal(tt.set.Len()+1, clone.Len())
			require.NotEqual(tt.set.AsArray(), clone.AsArray())
		})
	}
}

func TestSet_Contains(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		v    Month
		want bool
	}{
		{"empty", Set[Month]{}, Month_may, false},
		{"one", From(Month_may), Month_may, true},
		{"two", From(Month_may, Month_jun), Month_jun, true},
		{"negative", From(Month_may, Month_jun), Month_aug, false},
		{"out of bounds", From(Month_may, Month_jun, Month_count+1), Month_count + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Contains(tt.v); got != tt.want {
				t.Errorf("Set(%v).Contains(%v) = %v, want %v", tt.set, tt.v, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAll(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[Month]
		values []Month
		want   bool
	}{
		{"nil in empty", Set[Month]{}, nil, true},
		{"empty in empty", Set[Month]{}, []Month{}, true},
		{"cdoc in empty", Set[Month]{}, []Month{Month_may}, false},
		{"cdoc in cdoc", From(Month_may), []Month{Month_may}, true},
		{"cdoc + odoc in cdoc", From(Month_may), []Month{Month_may, Month_jun}, false},
		{"cdoc + odoc in cdoc + odoc", From(Month_may, Month_jun), []Month{Month_may, Month_jun}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAll(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAll(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_ContainsAny(t *testing.T) {
	tests := []struct {
		name   string
		set    Set[Month]
		values []Month
		want   bool
	}{
		{"nil in []", Set[Month]{}, nil, true},
		{"[] in []", Set[Month]{}, []Month{}, true},
		{"may in []", Set[Month]{}, []Month{Month_may}, false},
		{"may in [may]", From(Month_may), []Month{Month_may}, true},
		{"may, jun in [may]", From(Month_may), []Month{Month_may, Month_jun}, true},
		{"may, jun in [jul apr]", From(Month_jul, Month_apr), []Month{Month_may, Month_jun}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.ContainsAny(tt.values...); got != tt.want {
				t.Errorf("Set(%v).ContainsAny(%v) = %v, want %v", tt.set, tt.values, got, tt.want)
			}
		})
	}
}

func TestSet_First(t *testing.T) {
	tests := []struct {
		name      string
		set       Set[Month]
		want      bool
		wantValue Month
	}{
		{"empty", Set[Month]{}, false, Month_jan},
		{"one", From(Month_sep), true, Month_sep},
		{"two", From(Month_sep, Month_apr), true, Month_apr},
		{"out of bounds", From(Month_count + 1), true, Month_count + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotV := tt.set.First()
			if got != tt.want {
				t.Errorf("Set(%v).First() got = %v, want %v", tt.set, got, tt.want)
			}
			if !reflect.DeepEqual(gotV, tt.wantValue) {
				t.Errorf("Set(%v).First() gotV = %v, want %v", tt.set, gotV, tt.wantValue)
			}
		})
	}
}

func TestSet_Len(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		want int
	}{
		{"empty", Set[Month]{}, 0},
		{"one", From(Month_may), 1},
		{"two", From(Month_may, Month_feb), 2},
		{"two + out of bounds", From(Month_may, Month_oct, Month_count+1), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Len(); got != tt.want {
				t.Errorf("Set(%v).Len() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}
}

func TestSet_PutInt64(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		arg  uint64
		want string
	}{
		{"empty", From(Month_jan), 0, "[]"},
		{"one", From(Month_jan), 1 << Month_may, "[may]"},
		{"two", From(Month_jan), 1<<Month_may | 1<<Month_jun, "[may jun]"},
		{"out of bounds", From(Month_jan), 1<<Month_may | 1<<(Month_count+1), fmt.Sprintf("[may %v]", Month_count+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.set.PutInt64(tt.arg)
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set.PutInt64(%v).String() = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestSet_SetRange(t *testing.T) {
	type args struct {
		start Month
		end   Month
	}
	tests := []struct {
		name string
		set  Set[Month]
		args args
		want string
	}{
		{"empty", Set[Month]{}, args{Month_may, Month_may}, "[]"},
		{"one", Set[Month]{}, args{Month_may, Month_may + 1}, "[may]"},
		{"two", Set[Month]{}, args{Month_may, Month_may + 2}, "[may jun]"},
		{"three", Set[Month]{}, args{Month_may, Month_may + 3}, "[may jun jul]"},
		{"one + range", From(Month_jan), args{Month_may, Month_may + 3}, "[jan may jun jul]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.set.SetRange(tt.args.start, tt.args.end)
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set.SetRange(%v, %v).String() = %v, want %v", tt.args.start, tt.args.end, got, tt.want)
			}
		})
	}
}

type WeekDay uint8

const (
	WeekDay_mon WeekDay = iota
	WeekDay_tue
	WeekDay_wed
	WeekDay_thu
	WeekDay_fri
	WeekDay_sat
	WeekDay_sun
)

// func (t WeekDay) String() string {
// 	return fmt.Sprintf("WeekDay(%d)", t)
// }

func TestSet_String(t *testing.T) {
	tests := []struct {
		name string
		set  Set[Month]
		want string
	}{
		{"empty", Set[Month]{}, "[]"},
		{"one", From(Month_may), "[may]"},
		{"two", From(Month_may, Month_nov), "[may nov]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.String(); got != tt.want {
				t.Errorf("Set(%v).String() = %v, want %v", tt.set, got, tt.want)
			}
		})
	}

	t.Run("should render WeekDay", func(t *testing.T) {
		set := From(WeekDay_mon, WeekDay_fri)
		require.Equal(t, "[0 4]", set.String())
	})
}
