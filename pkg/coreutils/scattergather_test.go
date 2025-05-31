/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"
)

func TestScatterGatherBasic(t *testing.T) {
	src := []int{1, 2, 3, 4, 5}
	var got []int

	err := ScatterGather(context.Background(), src, 3,
		func(v int) (int, error) { return v * 2, nil },
		func(v int) { got = append(got, v) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{2, 4, 6, 8, 10}
	sort.Ints(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("results mismatch:\nwant %v\n got %v", want, got)
	}
}

func TestScatterGatherEmptySource(t *testing.T) {
	called := false
	err := ScatterGather(context.Background(), []int{}, 4,
		func(v int) (int, error) { return v, nil },
		func(int) { called = true },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("gatherer should not be called for empty source")
	}
}

func TestScatterGatherMapperError(t *testing.T) {
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
		func(int) { atomic.AddInt32(&gathered, 1) },
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	if g := atomic.LoadInt32(&gathered); g == int32(len(src)) {
		t.Fatalf("gatherer processed all elements despite error; gathered=%d", g)
	}
}

func TestScatterGatherContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := ScatterGather(ctx, []int{1, 2, 3}, 1,
		func(v int) (int, error) { return v, nil },
		func(int) {},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestScatterGatherWorkersZero(t *testing.T) {
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
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("results mismatch:\nwant %v\n got %v", want, got)
	}
}
