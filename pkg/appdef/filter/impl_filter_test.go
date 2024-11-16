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

func Test_filter(t *testing.T) {
	require := require.New(t)
	f := filter{}
	require.Nil(f.And(), "filter.And() should be nil")
	require.Equal(appdef.FilterKind_null, f.Kind(), "filter.Kind() should be null")
	require.Nil(f.Or(), "filter.Or() should be nil")
	require.Empty(f.QNames(), "filter.QNames() should be empty")
	require.False(f.Match(appdef.NullType), "filter.Match() should be false")
	require.Equal(NullResults, f.Matches(nil), "filter.Matches() should be null")
	require.Empty(f.Tags(), "filter.Tags() should be empty")
	require.Equal(appdef.TypeKindSet{}, f.Types(), "filter.Types() should be empty")
}

func Test_copyResults(t *testing.T) {
	require := require.New(t)

	r := makeResults()
	r.add(appdef.NullType)

	clone := copyResults(r)

	require.Equal(r.TypeCount(), clone.TypeCount(), "should be equal TypeCount()")
	require.Equal(fmt.Sprint(r), fmt.Sprint(clone), "should be equal String()")
}

func Test_results_String(t *testing.T) {
	tests := []struct {
		r    *results
		want string
	}{
		{makeResults(), "[]"},
		{makeResults(appdef.NullType), "[null type]"},
		{makeResults(appdef.NullType, appdef.AnyType), "[null type, ANY type]"},
	}

	require := require.New(t)
	for i, test := range tests {
		got := fmt.Sprint(test.r)
		require.Equal(test.want, got, "test %d", i)
	}
}

func Test_results_Type(t *testing.T) {
	require := require.New(t)

	r := makeResults(appdef.AnyType)

	require.Equal(appdef.AnyType, r.Type(appdef.QNameANY), "should find known type")
	require.Equal(appdef.NullType, r.Type(appdef.NewQName("test", "unknown")), "should null type if unknown")
}

func Test_results_TypeCount(t *testing.T) {
	tests := []struct {
		r    *results
		want int
	}{
		{makeResults(), 0},
		{makeResults(appdef.NullType), 1},
		{makeResults(appdef.NullType, appdef.AnyType), 2},
	}

	require := require.New(t)
	for i, test := range tests {
		got := test.r.TypeCount()
		require.Equal(test.want, got, "test %d", i)
	}
}

func Test_results_Types(t *testing.T) {
	tests := []struct {
		r    *results
		want []appdef.IType
	}{
		{makeResults(), []appdef.IType{}},
		{makeResults(appdef.NullType), []appdef.IType{appdef.NullType}},
		{makeResults(appdef.NullType, appdef.AnyType), []appdef.IType{appdef.NullType, appdef.AnyType}},
	}

	require := require.New(t)
	for i, test := range tests {
		got := []appdef.IType{}
		for t := range test.r.Types {
			got = append(got, t)
		}
		require.Equal(test.want, got, "test %d", i)
	}

	require.Equal(1, func() int {
		cnt := 0
		for range makeResults(appdef.NullType, appdef.AnyType).Types {
			cnt++
			break
		}
		return cnt
	}(), "should be breakable")
}
