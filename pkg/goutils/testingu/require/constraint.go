/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"fmt"
	"strings"

	"github.com/stretchr/testify/assert"
)

// Constraint is a common function prototype when validating given value.
type Constraint assert.ValueAssertionFunc

// Returns a constraint that checks that value (panic or error) contains
// the given substring.
func Has(substr interface{}, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		return assert.Contains(t, fmt.Sprint(v), fmt.Sprint(substr), msgAndArgs...)
	}
}

// Returns a constraint that checks that value (panic or error) contains
// all the given substrings.
func HasAll(substr ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		list := fmt.Sprint(v)
		for _, s := range substr {
			if !assert.Contains(t, list, fmt.Sprint(s)) {
				return false
			}
		}
		return true
	}
}

// Returns a constraint that checks that value (panic or error) contains
// at least one from the given substrings.
func HasAny(substr ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		if len(substr) == 0 {
			return true
		}

		list := fmt.Sprint(v)
		for _, s := range substr {
			if strings.Contains(list, fmt.Sprint(s)) {
				return true
			}
		}
		return assert.Contains(t, list, fmt.Sprint(substr...))
	}
}

// Returns a constraint that checks that value (panic or error) does not contain
// the given substring.
func NotHas(substr string, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		return assert.NotContains(t, fmt.Sprint(v), substr, msgAndArgs...)
	}
}

// Return constraint that checks if specified regexp matches value (panic or error).
func Rx(rx interface{}, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		return assert.Regexp(t, rx, v, msgAndArgs...)
	}
}

// Returns a constraint that checks that value (panic or error) does not match
// specified regexp.
func NotRx(rx interface{}, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		return assert.NotRegexp(t, rx, v, msgAndArgs...)
	}
}

// Returns a constraint that checks that error (or one of the errors in the error chain)
// matches the target.
func Is(target error, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		err, ok := v.(error)
		if !ok {
			return assert.Fail(t, fmt.Sprintf("«%#v» is not an error", v), msgAndArgs...)
		}
		return assert.ErrorIs(t, err, target, msgAndArgs...) //nolint:testifylint // Use of require inside require is inappropriate
	}
}

// Returns a constraint that checks that none of the errors in the error chain
// match the target.
func NotIs(target error, msgAndArgs ...interface{}) Constraint {
	return func(t assert.TestingT, v interface{}, _ ...interface{}) bool {
		err, ok := v.(error)
		if !ok {
			return true
		}
		return assert.NotErrorIs(t, err, target, msgAndArgs...) //nolint:testifylint // Use of require inside require is inappropriate
	}
}

// PanicsWith asserts that the code inside the specified function panics,
// and that the recovered panic value is satisfies the given constraints.
//
//	require.PanicsWith(t,
//		func(){ GoCrazy() },
//		require.Has("crazy"),
//		require.Rx("^.*\s+error$"))
func PanicsWith(t assert.TestingT, f func(), c ...Constraint) bool {
	return panicsWith(t, f, c)
}

func panicsWith(t assert.TestingT, f func(), c []Constraint, msgAndArgs ...interface{}) bool {
	didPanic := func() (wasPanic bool, recovered any) {
		defer func() {
			if recovered = recover(); recovered != nil {
				wasPanic = true
			}
		}()

		f()

		return wasPanic, recovered
	}

	wasPanic, recovered := didPanic()

	if !wasPanic {
		return assert.Fail(t, "panic expected", msgAndArgs...)
	}

	for _, constraint := range c {
		if !constraint(t, recovered) {
			return false
		}
	}

	return true
}

// ErrorWith asserts that the given error is not nil and satisfies the given constraints.
//
//	require.ErrorWith(t,
//		err,
//		require.Is(MyError),
//		require.Has("my message"))
func ErrorWith(t assert.TestingT, e error, c ...Constraint) bool {
	return errorWith(t, e, c)
}

func errorWith(t assert.TestingT, e error, c []Constraint, msgAndArgs ...interface{}) bool {
	if e == nil {
		return assert.Fail(t, "error expected", msgAndArgs)
	}

	for _, constraint := range c {
		if !constraint(t, e) {
			return false
		}
	}

	return true
}
