// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrPipeline_Unwrap(t *testing.T) {
	var testErr = errors.New("test")
	var err IErrorPipeline = &errPipeline{err: testErr}

	require.ErrorIs(t, err, testErr)
	require.NotErrorIs(t, err, errors.New("boom"))
}

func TestErrPipeline_Release(t *testing.T) {
	require.NotPanics(t, func() {
		errPipeline{}.Release()
	})
}

type err1 struct{}

func (e err1) Error() string {
	return ""
}

type err2 struct{}

func (e err2) Error() string {
	return ""
}

type err3 struct{}

func (e err3) Error() string {
	return ""
}

func TestErrInBranches(t *testing.T) {
	require := require.New(t)
	eib := ErrInBranches{Errors: []error{err1{}, err2{}, err3{}}}

	var e1 err1
	require.ErrorAs(eib, &e1)

	var e2 err2
	require.ErrorAs(eib, &e2)

	var e3 err3
	require.ErrorAs(eib, &e3)
}
