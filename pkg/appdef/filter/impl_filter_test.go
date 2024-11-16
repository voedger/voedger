/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package filter

import (
	"fmt"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func TestFilter_And(t *testing.T) {
	f := filter{}
	if got := f.And(); got != nil {
		t.Errorf("filter.And() = %v, want nil", got)
	}
}

func TestFilter_Kind(t *testing.T) {
	f := filter{}
	if got := f.Kind(); got != appdef.FilterKind_null {
		t.Errorf("filter.Kind() = %v, want %v", got, appdef.FilterKind_null)
	}
}

func TestFilter_Not(t *testing.T) {
	f := filter{}
	if got := f.Not(); got != nil {
		t.Errorf("filter.Not() = %v, want nil", got)
	}
}

func TestFilter_Or(t *testing.T) {
	f := filter{}
	if got := f.Or(); got != nil {
		t.Errorf("filter.Or() = %v, want nil", got)
	}
}

func TestFilter_QNames(t *testing.T) {
	f := filter{}
	if got := f.QNames(); got != nil {
		t.Errorf("filter.QNames() = %v, want nil", got)
	}
}

func TestFilter_Match(t *testing.T) {
	f := filter{}
	if got := f.Match(appdef.NullType); got != false {
		t.Errorf("filter.Match() = %v, want false", got)
	}
}

func TestFilter_Matches(t *testing.T) {
	f := filter{}
	if got := f.Matches(nil); got.TypeCount() != 0 {
		t.Errorf("filter.Matches() = %v, want empty", got)
	}
}

func TestFilter_Tags(t *testing.T) {
	f := filter{}
	if got := f.Tags(); got != nil {
		t.Errorf("filter.Tags() = %v, want nil", got)
	}
}

func TestFilter_Types(t *testing.T) {
	f := filter{}
	if got := f.Types(); got != (appdef.TypeKindSet{}) {
		t.Errorf("filter.Types() = %v, want %v", got, appdef.TypeKindSet{})
	}
}

func Test_cloneTypes(t *testing.T) {
	require := require.New(t)

	tt := makeTypes()
	tt.add(appdef.NullType)

	clone := cloneTypes(tt)

	require.Equal(tt.TypeCount(), clone.TypeCount(), "should be equal TypeCount()")
	require.Equal(fmt.Sprint(tt), fmt.Sprint(clone), "should be equal String()")
}

func Test_typesString(t *testing.T) {
	tests := []struct {
		tt   *types
		want string
	}{
		{makeTypes(), "[]"},
		{makeTypes(appdef.NullType), "[null type]"},
		{makeTypes(appdef.NullType, appdef.AnyType), "[null type, ANY type]"},
	}

	require := require.New(t)
	for i, test := range tests {
		got := fmt.Sprint(test.tt)
		require.Equal(test.want, got, "test %d", i)
	}
}

func Test_typesType(t *testing.T) {
	require := require.New(t)

	tt := makeTypes(appdef.AnyType)

	require.Equal(appdef.AnyType, tt.Type(appdef.QNameANY), "should find known type")
	require.Equal(appdef.NullType, tt.Type(appdef.NewQName("test", "unknown")), "should null type if unknown")
}

func Test_typesTypeCount(t *testing.T) {
	tests := []struct {
		tt   *types
		want int
	}{
		{makeTypes(), 0},
		{makeTypes(appdef.NullType), 1},
		{makeTypes(appdef.NullType, appdef.AnyType), 2},
	}

	require := require.New(t)
	for i, test := range tests {
		got := test.tt.TypeCount()
		require.Equal(test.want, got, "test %d", i)
	}
}

func Test_typesTypes(t *testing.T) {
	tests := []struct {
		tt   *types
		want []appdef.IType
	}{
		{makeTypes(), []appdef.IType{}},
		{makeTypes(appdef.NullType), []appdef.IType{appdef.NullType}},
		{makeTypes(appdef.NullType, appdef.AnyType), []appdef.IType{appdef.NullType, appdef.AnyType}},
	}

	require := require.New(t)
	for i, test := range tests {
		got := []appdef.IType{}
		for t := range test.tt.Types {
			got = append(got, t)
		}
		require.Equal(test.want, got, "test %d", i)
	}

	require.Equal(1, func() int {
		cnt := 0
		for range makeTypes(appdef.NullType, appdef.AnyType).Types {
			cnt++
			break
		}
		return cnt
	}(), "should be breakable")
}
