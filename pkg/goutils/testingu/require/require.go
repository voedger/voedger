/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type Require struct {
	*require.Assertions
	t *testing.T
}

func New(t *testing.T) *Require {
	return &Require{
		Assertions: require.New(t),
		t:          t,
	}
}

// Returns a constraint that checks that value (panic or error) contains
// the given substring.
func (r *Require) Has(substr string, msgAndArgs ...interface{}) Constraint {
	return Has(substr, msgAndArgs...)
}

// Returns a constraint that checks that value (panic or error) does not contain
// the given substring.
func (r *Require) NotHas(substr string, msgAndArgs ...interface{}) Constraint {
	return NotHas(substr, msgAndArgs...)
}

// Returns a constraint that checks that value (panic or error) matches
// specified regexp.
func (r *Require) Rx(rx interface{}, msgAndArgs ...interface{}) Constraint {
	return Rx(rx, msgAndArgs...)
}

// Returns a constraint that checks that value (panic or error) does not match
// specified regexp.
func (r *Require) NotRx(rx interface{}, msgAndArgs ...interface{}) Constraint {
	return NotRx(rx, msgAndArgs...)
}

// Returns a constraint that checks that error (or one of the errors in the error chain)
// matches the target.
func (r *Require) Is(targer error, msgAndArgs ...interface{}) Constraint {
	return Is(targer, msgAndArgs...)
}

// Returns a constraint that checks that none of the errors in the error chain
// match the target.
func (r *Require) NotIs(targer error, msgAndArgs ...interface{}) Constraint {
	return NotIs(targer, msgAndArgs...)
}

// PanicsWith asserts that the code inside the specified function panics,
// and that the recovered panic value is satisfies the given constraints.
//
//	require := require.New(t)
//	require.PanicsWith(
//		func(){ GoCrazy() },
//		require.Has("crazy"),
//		require.NotHas("smile"),
//		require.Rx("^.*\s+error$"))
func (r *Require) PanicsWith(f func(), c ...Constraint) {
	if !PanicsWith(r.t, f, c...) {
		r.t.FailNow()
	}
}

// ErrorWith asserts that the given error is not nil and satisfies the given constraints.
//
//	require := require.New(t)
//	require.ErrorWith(
//		err,
//		require.Is(MyError),
//		require.Has("my message"))
func (r *Require) ErrorWith(e error, c ...Constraint) {
	if !ErrorWith(r.t, e, c...) {
		r.t.FailNow()
	}
}
