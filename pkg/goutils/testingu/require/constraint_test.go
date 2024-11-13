/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
)

func TestPanicsWith(t *testing.T) {
	testError := fmt.Errorf("my test error: %w", errors.ErrUnsupported)
	tests := []struct {
		name string
		f    func()
		c    Constraint
		want bool
	}{
		{"Should fail if no expected panic",
			func() {}, Has("test"), false},
		// Has ———————————————————————————————————————
		{"Should be ok if panic occurs and contains expected message",
			func() { panic("my crazy message") }, Has("crazy"), true},
		{"Should be ok if panic occurs and contains error with expected message",
			func() { panic(testError) }, Has("test"), true},
		{"Should fail if panic with unexpected message",
			func() { panic("other error") }, Has("test"), false},
		// HasAll ———————————————————————————————————————
		{"Should be ok if panic occurs and contains all expected messages",
			func() { panic("my crazy message") }, HasAll("my", "crazy", "message"), true},
		{"Should be ok if panic occurs and contains error with all expected message",
			func() { panic(testError) }, HasAll("my", "test", "error"), true},
		{"Should fail if panic not with all unexpected messages",
			func() { panic("other error") }, HasAll("test", "error"), false},
		// HasAny ———————————————————————————————————————
		{"Should be ok if panic occurs and contains any of expected messages",
			func() { panic("my crazy message") }, HasAny("hot", "crazy", "🔫"), true},
		{"Should be ok if panic occurs and contains error with all expected message",
			func() { panic(testError) }, HasAny("hot", "test", "🔫"), true},
		{"Should fail if panic not contains anyone from unexpected messages",
			func() { panic("other error") }, HasAny("hot", "crazy", "🔫"), false},
		// NotHas ———————————————————————————————————————
		{"Should be ok if panic occurs and does contains deprecated message",
			func() { panic("my crazy message") }, NotHas("deprecated"), true},
		{"Should be ok if panic occurs and does not contains error with deprecated message",
			func() { panic(testError) }, NotHas("deprecated"), true},
		{"Should fail if panic with deprecated message",
			func() { panic("deprecated error") }, NotHas("deprecated"), false},
		// Rx ————————————————————————————————————————
		{"Should be ok if regexp (string) matches panic",
			func() { panic(testError) }, Rx("^my test error"), true},
		{"Should be ok if regexp (compiled) matches panic",
			func() { panic(testError) }, Rx(regexp.MustCompile("^my test error")), true},
		{"Should fail if regexp does not matches panic",
			func() { panic(errors.New("other error")) }, Rx("^my test error"), false},
		// NotRx ————————————————————————————————————————
		{"Should be ok if regexp (string) does not matches panic",
			func() { panic(testError) }, NotRx("deprecated"), true},
		{"Should be ok if regexp (compiled) matches panic",
			func() { panic(testError) }, NotRx(regexp.MustCompile("deprecated")), true},
		{"Should fail if regexp matches panic",
			func() { panic(testError) }, NotRx(`my\s+test`), false},
		// Is ————————————————————————————————————————
		{"Should be ok if panic occurs with expected error",
			func() { panic(testError) }, Is(testError), true},
		{"Should be ok if panic occurs with expected error in chain",
			func() { panic(fmt.Errorf("%w: test", testError)) }, Is(errors.ErrUnsupported), true},
		{"Should fail if panic without error",
			func() { panic("panic message") }, Is(testError), false},
		{"Should fail if panic with other error",
			func() { panic(errors.New("other error")) }, Is(testError), false},
		// NotIs ————————————————————————————————————————
		{"Should be ok if panic occurs with other error",
			func() { panic(errors.New("other error")) }, NotIs(testError), true},
		{"Should be ok if panic without error",
			func() { panic("panic") }, NotIs(testError), true},
		{"Should fail if panic with expected error",
			func() { panic(testError) }, NotIs(testError), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := PanicsWith(mockT, tt.f, tt.c)
			if r != tt.want {
				t.Errorf("PanicsWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}

func TestErrorWith(t *testing.T) {
	testError := fmt.Errorf("my test error: %w", errors.ErrUnsupported)
	tests := []struct {
		name string
		e    error
		c    Constraint
		want bool
	}{
		{"Should fail if no error",
			nil, Has("my test"), false},
		// Has ———————————————————————————————————————
		{"Should be ok if error occurs and contains expected message",
			testError, Has("my test"), true},
		{"Should be ok if error occurs and contains error which wraps expected error",
			fmt.Errorf("wrapped: %w", testError), Has("my test"), true},
		{"Should fail if unexpected error",
			errors.New("other error"), Has("my test"), false},
		// HasAll ———————————————————————————————————————
		{"Should be ok if error occurs and contains all expected messages",
			testError, HasAll("my", "test"), true},
		{"Should be ok if error occurs and contains error which wraps error with all expected messages",
			fmt.Errorf("wrapped: %w", testError), HasAll("wrapped", "my", "test"), true},
		{"Should fail if error not contains all expected messages",
			errors.New("other error"), HasAll("my", "error"), false},
		// HasAny ———————————————————————————————————————
		{"Should be ok if error occurs and contains any from expected messages",
			testError, HasAny("some", "test"), true},
		{"Should be ok if error occurs and contains error which wraps error with any from expected messages",
			fmt.Errorf("wrapped: %w", testError), HasAny("some", "test"), true},
		{"Should fail if error not contains anyone from expected messages",
			errors.New("other error"), HasAny("hot", "crazy", "🔫"), false},
		{"Should be ok if error occurs and expected messages list is empty",
			testError, HasAny(), true},
		// NotHas ———————————————————————————————————————
		{"Should be ok if error occurs and does not contains deprecated message",
			testError, NotHas("deprecated"), true},
		{"Should fail if error with deprecated message",
			errors.New("deprecated error"), NotHas("deprecated"), false},
		{"Should fail if error with deprecated wrap",
			fmt.Errorf("deprecated: %w", testError), NotHas("deprecated"), false},
		// Rx ————————————————————————————————————————
		{"Should be ok if regexp (string) mathes error",
			testError, Rx("unsupported operation$"), true},
		{"Should be ok if regexp (compiled) mathes error",
			testError, Rx(regexp.MustCompile("^my test error")), true},
		{"Should fail if regexp does not mathes error",
			errors.New("other error"), Rx("my"), false},
		// NotRx ————————————————————————————————————————
		{"Should be ok if regexp (string) does not mathes error",
			testError, NotRx("deprecated"), true},
		{"Should be ok if regexp (compiled) mathes error",
			testError, NotRx(regexp.MustCompile("deprecated")), true},
		{"Should fail if regexp mathes error",
			testError, NotRx(`my\s+test`), false},
		// Is ————————————————————————————————————————
		{"Should be ok if error occurs with expected error",
			testError, Is(testError), true},
		{"Should be ok if error occurs with expected error in chain",
			fmt.Errorf("%w: test", testError), Is(errors.ErrUnsupported), true},
		{"Should fail if error with other error",
			errors.New("other error"), Is(testError), false},
		// NotIs ————————————————————————————————————————
		{"Should be ok if error occurs with other error",
			errors.New("other error"), NotIs(testError), true},
		{"Should fail if error with expected error",
			testError, NotIs(errors.ErrUnsupported), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := ErrorWith(mockT, tt.e, tt.c)
			if r != tt.want {
				t.Errorf("ErrorWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}
