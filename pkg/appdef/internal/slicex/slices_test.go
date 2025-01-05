/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package slicex_test

import (
	"testing"

	"github.com/voedger/voedger/pkg/appdef/internal/slicex"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
)

func Test_Duplicates(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		wantI int
		wantJ int
	}{
		{
			name:  "empty slice",
			slice: []int{},
			wantI: -1,
			wantJ: -1,
		},
		{
			name:  "no duplicates",
			slice: []int{1, 2, 3},
			wantI: -1,
			wantJ: -1,
		},
		{
			name:  "duplicates",
			slice: []int{1, 2, 1},
			wantI: 0,
			wantJ: 2,
		},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, j := slicex.FindDuplicates(tt.slice)
			require.Equal(tt.wantI, i)
			require.Equal(tt.wantJ, j)
		})
	}
}

func Test_InsertInSort(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		v    int
		want []int
	}{
		{
			name: "empty slice",
			s:    []int{},
			v:    1,
			want: []int{1},
		},
		{
			name: "insert in the beginning",
			s:    []int{2, 3},
			v:    1,
			want: []int{1, 2, 3},
		},
		{
			name: "insert in the middle",
			s:    []int{1, 3},
			v:    2,
			want: []int{1, 2, 3},
		},
		{
			name: "insert in the end",
			s:    []int{1, 2},
			v:    3,
			want: []int{1, 2, 3},
		},
		{
			name: "insert existing value",
			s:    []int{1, 2, 3},
			v:    2,
			want: []int{1, 2, 3},
		},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, slicex.InsertInSort(tt.s, tt.v, func(i, j int) int { return i - j }))
		})
	}
}

func Test_SubSet(t *testing.T) {
	tests := []struct {
		name string
		sub  []int
		set  []int
		want bool
	}{
		{
			name: "empty slices",
			sub:  []int{},
			set:  []int{},
			want: true,
		},
		{
			name: "nil sub slice",
			sub:  nil,
			set:  []int{1, 2, 3},
			want: true,
		},
		{
			name: "nil set slice",
			sub:  []int{},
			set:  nil,
			want: true,
		},
		{
			name: "nil slices",
			sub:  nil,
			set:  nil,
			want: true,
		},
		{
			name: "sub is subset of set",
			sub:  []int{1},
			set:  []int{1, 2, 3},
			want: true,
		},
		{
			name: "sub is not subset of set",
			sub:  []int{1},
			set:  []int{2, 3},
			want: false,
		},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, slicex.IsSubSet(tt.sub, tt.set))
		})
	}
}

func Test_Overlaps(t *testing.T) {
	tests := []struct {
		name string
		set1 []int
		set2 []int
		want bool
	}{
		{
			name: "empty slices",
			set1: []int{},
			set2: []int{},
			want: true,
		},
		{
			name: "nil set1 slice",
			set1: nil,
			set2: []int{1, 2, 3},
			want: true,
		},
		{
			name: "nil set2 slice",
			set1: []int{1, 2, 3},
			set2: nil,
			want: true,
		},
		{
			name: "nil slices",
			set1: nil,
			set2: nil,
			want: true,
		},
		{
			name: "set1 is subset of set2",
			set1: []int{2},
			set2: []int{1, 2, 3},
			want: true,
		},
		{
			name: "set2 is subset of set1",
			set1: []int{1, 2, 3},
			set2: []int{1, 3},
			want: true,
		},
		{
			name: "set1 and set2 are not overlapped",
			set1: []int{1, 3, 5},
			set2: []int{2, 4, 6},
			want: false,
		},
	}

	require := require.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(tt.want, slicex.Overlaps(tt.set1, tt.set2))
		})
	}
}
