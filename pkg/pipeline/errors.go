/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

import (
	"errors"
	"strings"
)

type IErrorPipeline interface {
	error
	IWorkpiece
	GetWork() interface{}
}

type errPipeline struct {
	err  error
	work interface{}
}

func (e errPipeline) Release() {
}

func (e errPipeline) Error() string {
	return e.err.Error()
}

func (e errPipeline) Unwrap() error {
	return e.err
}

func (e errPipeline) GetWork() interface{} {
	return e.work
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
