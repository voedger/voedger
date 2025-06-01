/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScatterGatherBasic(t *testing.T) {
	require := require.New(t)
	src := []int{1, 2, 3, 4, 5}
	var got []int

	err := ScatterGather(context.Background(), src, 3,
		func(v int) (int, error) { return v * 2, nil },
		func(v int) { got = append(got, v) },
	)
	require.NoError(err)

	want := []int{2, 4, 6, 8, 10}
	sort.Ints(got)
	require.Equal(want, got)
}

func TestScatterGatherEmptySource(t *testing.T) {
	require := require.New(t)
	called := false
	err := ScatterGather(context.Background(), []int{}, 4,
		func(v int) (int, error) { return v, nil },
		func(int) { called = true },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	require.False(called)
}

func TestScatterGatherMapperError(t *testing.T) {
	require := require.New(t)
	src := []int{1, 2, 3, 4, 5}
	sentinel := errors.New("boom")
	var gathered int32

	err := ScatterGather(context.Background(), src, 2,
		func(v int) (int, error) {
			if v == 3 {
				return 0, sentinel
			}
			return v, nil
		},
		func(int) { gathered++ },
	)

	require.ErrorIs(err, sentinel)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	require.NotEqual(len(src), gathered)
}

func TestScatterGatherContextCancel(t *testing.T) {
	require := require.New(t)

	t.Run("initially canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		err := ScatterGather(ctx, []int{1, 2, 3}, 1,
			func(v int) (int, error) { return v, nil },
			func(int) {},
		)

		require.ErrorIs(err, context.Canceled)
	})

	t.Run("cancel on mapping", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		err := ScatterGather(ctx, []int{1, 2, 3}, 1,
			func(v int) (int, error) {
				cancel()
				return v, nil
			},
			func(int) {},
		)
		require.ErrorIs(err, context.Canceled)
	})

	t.Run("cancel on gathering", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		err := ScatterGather(ctx, []int{1, 2, 3}, 1,
			func(v int) (int, error) { return v, nil },
			func(int) { cancel() },
		)
		require.ErrorIs(err, context.Canceled)
	})

	t.Run("mapper error priority over context cancel", func(t *testing.T) {
		mapperError := errors.New("boom")
		ctx, cancel := context.WithCancel(context.Background())
		err := ScatterGather(ctx, []int{1, 2, 3}, 1,
			func(v int) (int, error) {
				cancel()
				return v, mapperError
			},
			func(int) {},
		)
		require.ErrorIs(err, mapperError)
	})
}

func TestScatterGatherWorkersZero(t *testing.T) {
	require := require.New(t)
	src := []string{"a", "b", "c"}
	got := make([]string, 0, len(src))

	if err := ScatterGather(context.Background(), src, 0,
		func(s string) (string, error) { return s + s, nil },
		func(s string) { got = append(got, s) },
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(got)
	want := []string{"aa", "bb", "cc"}
	require.Equal(want, got)
}
