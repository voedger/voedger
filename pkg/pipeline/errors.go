// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko
// @author Alisher Nurmanov

package pipeline

import (
	"errors"
	"strings"
)

type IErrorPipeline interface {
	error
	IWorkpiece
	GetWork() IWorkpiece
	GetOpName() string
	GetPlace() string
}

type errPipeline struct {
	err    error
	work   IWorkpiece
	place  string
	opName string
}

func (e errPipeline) Release() {
}

func (e errPipeline) Error() string {
	return e.err.Error()
}

func (e errPipeline) Unwrap() error {
	return e.err
}

func (e errPipeline) GetWork() IWorkpiece {
	return e.work
}

func (e errPipeline) GetOpName() string {
	return e.opName
}

func (e errPipeline) GetPlace() string {
	return e.place
}

type ErrInBranches struct {
	Errors []error
}

func (e ErrInBranches) Error() string {
	ss := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		ss[i] = err.Error()
	}
	return strings.Join(ss, ",")
}

// need for uniques projector at ackages/sys/uniques/impl.go
// it emmits 409 Conflict HTTP status code, so need to pull it from ErrInBranches
func (e ErrInBranches) As(target interface{}) bool {
	for _, err := range e.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
