/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Constraint is a common function prototype when validating given value.
type Constraint func(*testing.T, interface{}) bool

// Return constraint that checks if value (panic or error) contains given substring.
func Has(substr string, msgAndArgs ...interface{}) Constraint {
	return func(t *testing.T, recovered interface{}) bool {
		m := fmt.Sprint(recovered)
		return assert.Contains(t, m, substr, msgAndArgs...)
	}
}

// Return constraint that checks if value is error (or errors chain) and at least one of the errors
// in err's chain matches target.
func Is(target error, msgAndArgs ...interface{}) Constraint {
	return func(t *testing.T, err interface{}) bool {
		e, ok := err.(error)
		if !ok {
			return assert.Fail(t, fmt.Sprintf("«%#v» is not an error", err), msgAndArgs...)
		}
		return assert.ErrorIs(t, e, target, msgAndArgs...) //nolint:testifylint // Use of require inside require is inappropriate
	}
}

// PanicsWith asserts that the code inside the specified function panics,
// and that the recovered panic value is satisfies the given constraints.
//
//	require.PanicsWith(t,
//		func(){ GoCrazy() },
//		require.Contains("crazy"),
//		require.Contains("error))
func PanicsWith(t *testing.T, f func(), c ...Constraint) bool {
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
		return assert.Fail(t, "panic expected")
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
//		require.Contains("my message"))
func ErrorWith(t *testing.T, e error, c ...Constraint) bool {
	if e == nil {
		return assert.Fail(t, "error expected")
	}

	for _, constraint := range c {
		if !constraint(t, e) {
			return false
		}
	}

	return true
}
