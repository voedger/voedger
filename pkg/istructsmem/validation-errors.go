/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import "fmt"

// validate error codes, see ValidateError.Code()
const (
	ECode_UnknownError = iota

	ECode_EmptyTypeName
	ECode_InvalidTypeName
	ECode_InvalidTypeKind

	ECode_EmptyData

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

const (
	// These errors are possible while checking raw identifiers specified in the event arguments and CUDs
	errRepeatedID                = "%v repeatedly uses record ID «%d» in %v: %w"
	errRequiredRawID             = "%v should use raw record ID (not «%d») in created %v: %w"
	errUnexpectedRawID           = "%v unexpectedly uses raw record ID «%d» in updated %v: %w"
	errRepeatedSingletonCreation = "%v repeatedly creates the same singleton %v (raw record ID «%d» and «%d»): %w"
	errUnknownIDRef              = "%v field «%s» refers to unknown record ID «%d»: %w"
	errUnavailableTargetRef      = "%v field «%s» refers to record ID «%d» that has unavailable target QName «%s»: %w"
	errParentHasNoContainer      = "%v has parent ID «%d» refers to «%s», which has no container «%s»: %w"
	errParentContainerOtherType  = "%v has parent ID «%d» refers to «%s», which container «%s» has another QName «%s»: %w"
)

const (
	// These errors are possible while checking type and content of the event arguments and CUDs
	errEventArgUseWrongType         = "%v argument uses wrong type «%v», expected «%v»: %w"
	errEventUnloggedArgUseWrongType = "%v unlogged argument uses wrong type «%v», expected «%v»: %w"
	errContainerMinOccursViolated   = "%v container «%s» has not enough occurrences (%d, minimum %d): %w"
	errContainerMaxOccursViolated   = "%v container «%s» has too many occurrences (%d, maximum %d): %w"
	errUnknownContainerName         = "%v child[%d] has unknown container name «%s»: %w"
	errWrongContainerType           = "%v child[%d] %v has wrong type name, expected «%v»: %w"
	errWrongParentID                = "%v child[%d] %v has wrong parent id «%d», expected «%d»: %w"
	errEmptyRequiredField           = "%v misses required field «%s»: %w"
	errNullInRequiredRefField       = "%v required ref field «%s» has NullRecordID value: %w"
	errCUDsMissed                   = "%v must have not empty CUDs: %w"
	errInvalidTypeKindInCUD         = "%v CUD.%s() [record ID «%d»] %v has invalid type kind: %w"
)
