/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"errors"
	"fmt"
	"testing"
)

func TestPanicsWithContains(t *testing.T) {
	tests := []struct {
		name string
		f    func()
		want bool
	}{
		{"Should be ok if panic occurs and contains expected message",
			func() { panic("crazy error") }, true},
		{"Should be ok if panic occurs and contains error with expected message",
			func() { panic(errors.New("crazy error")) }, true},
		{"Should fail if no expected panic",
			func() {}, false},
		{"Should fail if panic with unexpected message",
			func() { panic("other error") }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := PanicsWith(mockT, tt.f, Has("crazy"), Has("error"))
			if r != tt.want {
				t.Errorf("PanicsWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}

func TestPanicsWithIs(t *testing.T) {
	testError := errors.New("my test error")
	tests := []struct {
		name string
		f    func()
		want bool
	}{
		{"Should be ok if panic occurs with expected error",
			func() { panic(testError) }, true},
		{"Should fail if no expected panic",
			func() {}, false},
		{"Should fail if panic without error",
			func() { panic("panic message") }, false},
		{"Should fail if panic with other error",
			func() { panic(errors.New("other error")) }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := PanicsWith(mockT, tt.f, Is(testError))
			if r != tt.want {
				t.Errorf("PanicsWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}

func TestErrorWithContains(t *testing.T) {
	testError := errors.New("my test error")
	tests := []struct {
		name string
		e    error
		want bool
	}{
		{"Should be ok if error occurs and contains expected message",
			testError, true},
		{"Should be ok if error occurs and contains error which wraps expected error",
			fmt.Errorf("wrapped: %w", testError), true},
		{"Should fail if no error",
			nil,
			false,
		},
		{"Should fail if unexpected error",
			errors.New("other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := ErrorWith(mockT, tt.e, Has("my test"), Has("error"))
			if r != tt.want {
				t.Errorf("ErrorWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}

func TestErrorWithIs(t *testing.T) {
	testError := errors.New("my test error")
	tests := []struct {
		name string
		e    error
		want bool
	}{
		{"Should be ok if expected error",
			testError, true},
		{"Should be ok if error which wraps expected error",
			fmt.Errorf("wrapped: %w", testError), true},
		{"Should fail if no error",
			nil,
			false,
		},
		{"Should fail if other error",
			errors.New("other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := new(testing.T)
			r := ErrorWith(mockT, tt.e, Is(testError))
			if r != tt.want {
				t.Errorf("ErrorWith() returns %v, want %v", r, tt.want)
			}
		})
	}
}
