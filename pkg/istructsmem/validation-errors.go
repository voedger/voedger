/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import "fmt"

// validate error codes, see ValidateError.Code()
const (
	ECode_UnknownError = iota

	ECode_EmptySchemaName
	ECode_InvalidSchemaName
	ECode_InvalidDefKind

	ECode_EmptyData

	ECode_InvalidRawRecordID
	ECode_InvalidRecordID
	ECode_InvalidRefRecordID

	ECode_EEmptyCUDs

	ECode_EmptyElementName
	ECode_InvalidElementName
	ECode_InvalidOccursMin
	ECode_InvalidOccursMax
)

type validateErrorType struct {
	error
	code int
}

func (e validateErrorType) Code() int {
	return e.code
}

func (e validateErrorType) Unwrap() error {
	return e.error
}

func validateError(code int, err error) ValidateError {
	e := validateErrorType{
		error: fmt.Errorf("%w; validate error code: %d", err, code),
		code:  code,
	}
	return e
}

func validateErrorf(code int, format string, a ...interface{}) ValidateError {
	return validateError(code, fmt.Errorf(format, a...))
}
