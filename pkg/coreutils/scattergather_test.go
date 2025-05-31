package coreutils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestScatterGather(t *testing.T) {
	t.Run("successful processing", func(t *testing.T) {
		ctx := context.Background()
		input := []int{1, 2, 3, 4, 5}
		expected := []int{2, 4, 6, 8, 10}
		var results []int

		err := ScatterGather(ctx, input, 2,
			func(val int) (int, error) {
				return val * 2, nil
			},
			func(val int) {
				results = append(results, val)
			},
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(results) != len(expected) {
			t.Errorf("expected %d results, got %d", len(expected), len(results))
		}

		for i, v := range results {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("error handling", func(t *testing.T) {
		ctx := context.Background()
		input := []int{1, 2, 3}
		expectedErr := errors.New("processing error")

		err := ScatterGather(ctx, input, 2,
			func(val int) (int, error) {
				if val == 2 {
					return 0, expectedErr
				}
				return val * 2, nil
			},
			func(val int) {},
		)

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		input := make([]int, 1000)
		for i := range input {
			input[i] = i
		}

		err := ScatterGather(ctx, input, 4,
			func(val int) (int, error) {
				time.Sleep(50 * time.Millisecond)
				return val * 2, nil
			},
			func(val int) {},
		)

		if err != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		ctx := context.Background()
		var results []int

		err := ScatterGather(ctx, []int{}, 2,
			func(val int) (int, error) {
				return val * 2, nil
			},
			func(val int) {
				results = append(results, val)
			},
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected no results, got %d", len(results))
		}
	})

	t.Run("single worker", func(t *testing.T) {
		ctx := context.Background()
		input := []int{1, 2, 3, 4, 5}
		expected := []int{2, 4, 6, 8, 10}
		var results []int

		err := ScatterGather(ctx, input, 1,
			func(val int) (int, error) {
				return val * 2, nil
			},
			func(val int) {
				results = append(results, val)
			},
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(results) != len(expected) {
			t.Errorf("expected %d results, got %d", len(expected), len(results))
		}

		for i, v := range results {
			if v != expected[i] {
				t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
			}
		}
	})

	t.Run("concurrent processing order", func(t *testing.T) {
		ctx := context.Background()
		input := make([]int, 100)
		for i := range input {
			input[i] = i
		}

		processingOrder := make([]int, 0, 100)

		err := ScatterGather(ctx, input, 4,
			func(val int) (int, error) {
				time.Sleep(time.Duration(val%10) * time.Millisecond)
				return val, nil
			},
			func(val int) {
				processingOrder = append(processingOrder, val)
			},
		)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(processingOrder) != len(input) {
			t.Errorf("expected %d results, got %d", len(input), len(processingOrder))
		}

		// Verify that all values are present
		seen := make(map[int]bool)
		for _, v := range processingOrder {
			seen[v] = true
		}
		for i := 0; i < len(input); i++ {
			if !seen[i] {
				t.Errorf("value %d was not processed", i)
			}
		}
	})
}
