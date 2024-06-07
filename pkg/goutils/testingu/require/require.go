/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package require

import (
	"github.com/stretchr/testify/require"
)

type Require struct {
	*require.Assertions
	t require.TestingT
}

func New(t require.TestingT) *Require {
	return &Require{
		Assertions: require.New(t),
		t:          t,
	}
}

// Returns a constraint that checks that value (panic or error) contains
// the given substring.
func (r *Require) Has(substr interface{}, msgAndArgs ...interface{}) Constraint {
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

// Panics asserts that the code inside the specified PanicTestFunc panics.
// If constaintsAndMsgAndArgs contains constraints, then PanicsWith will be
// called with these constraints, else Panics will be called from testify/assert.
//
//	require := require.New(t)
//	require.Panics(
//		func(){ GoCrazy() },
//		require.Has("crazy", "panic should contains crazy string %q", "crazy"),
//		"crazy panic expected")
func (r *Require) Panics(f func(), constaintsOrMsgAndArgs ...interface{}) {
	cc := make([]Constraint, 0)
	for _, v := range constaintsOrMsgAndArgs {
		if c, ok := v.(Constraint); ok {
			cc = append(cc, c)
		}
	}
	if len(cc) > 0 {
		PanicsWith(r.t, f, cc...)
	} else {
		r.Assertions.Panics(f, constaintsOrMsgAndArgs...)
	}
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

// Error asserts that a function returned an error (i.e. not `nil`).
//
// If constaintsAndMsgAndArgs contains constraints, then ErrorWith will be
// called with these constraints, else Error will be called from testify/assert.
//
//	require := require.New(t)
//	result, err := SomeFunction()
//	require.Error(err,
//		require.Is(MyError),
//		require.Has("my message"))
func (r *Require) Error(err error, constaintsOrMsgAndArgs ...interface{}) {
	cc := make([]Constraint, 0)
	for _, v := range constaintsOrMsgAndArgs {
		if c, ok := v.(Constraint); ok {
			cc = append(cc, c)
		}
	}
	if len(cc) > 0 {
		ErrorWith(r.t, err, cc...)
	} else {
		r.Assertions.Error(err, constaintsOrMsgAndArgs...)
	}
}

// ErrorWith asserts that the given error is not nil and satisfies the given constraints.
//
//	require := require.New(t)
//	require.ErrorWith(err,
//		require.Is(MyError),
//		require.Has("my message"))
func (r *Require) ErrorWith(e error, c ...Constraint) {
	if !ErrorWith(r.t, e, c...) {
		r.t.FailNow()
	}
}
