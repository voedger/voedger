/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ibus

import (
	"context"
	"net/http"
)

func NewResult(response interface{}, err error, errMsg string, errData string) (resp interface{}, status Status, e error) {
	if err == nil {
		return response, Status{HTTPStatus: http.StatusOK}, nil
	}

	httpStatus, ok := ErrStatuses[err]
	if !ok {
		httpStatus = http.StatusInternalServerError
	}
	status = Status{
		HTTPStatus:   httpStatus,
		ErrorMessage: errMsg,
		ErrorData:    errData,
	}
	return response, status, err
}

func NullHandler(interface{}) {}

func EchoReceiver(_ context.Context, request interface{}, _ SectionsWriterType) (response interface{}, status Status, err error) {
	return NewResult(request, nil, "", "")
}
