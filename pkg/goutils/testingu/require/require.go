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

// Return requirement that checks if value (panic or error) contains given substring.
func (r *Require) Has(substr string, msgAndArgs ...interface{}) Constraint {
	return Has(substr, msgAndArgs...)
}

// Return constraint that checks if value is error (or errors chain) and at least one of the errors
// in err's chain matches target.
func (r *Require) Is(targer error, msgAndArgs ...interface{}) Constraint {
	return Is(targer, msgAndArgs...)
}

// PanicsWith asserts that the code inside the specified function panics,
// and that the recovered panic value is satisfies the given constraints.
//
//	require.PanicsWith(
//		func(){ GoCrazy() },
//		require.Contains("crazy"),
//		require.Contains("error))
func (r *Require) PanicsWith(f func(), c ...Constraint) {
	if !PanicsWith(r.t, f, c...) {
		r.t.FailNow()
	}
}

// ErrorWith asserts that the given error is not nil and satisfies the given constraints.
//
//	require.ErrorWith(
//		err,
//		require.Is(MyError),
//		require.Contains("my message"))
func (r *Require) ErrorWith(e error, c ...Constraint) {
	if !ErrorWith(r.t, e, c...) {
		r.t.FailNow()
	}
}
