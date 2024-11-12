/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"errors"
	"fmt"
)

func enrichError(err error, msg string, args ...any) error {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%w: %s", err, s)
}

// TODO: use enrichError() for all errors
// eliminate all calls fmt.Errorf("… %w …", …) with err×××Wrap constants

var ErrorEventNotValidError = errors.New("event is not valid")

func ErrorEventNotValid(msg string, args ...any) error {
	return enrichError(ErrorEventNotValidError, msg, args...)
}

var ErrNameMissedError = errors.New("name is empty")

func ErrNameMissed(msg string, args ...any) error {
	return enrichError(ErrNameMissedError, msg, args...)
}

var ErrOutOfBoundsError = errors.New("out of bounds")

func ErrOutOfBounds(msg string, args ...any) error {
	return enrichError(ErrOutOfBoundsError, msg, args...)
}

var ErrWrongTypeError = errors.New("wrong type")

func ErrWrongType(msg string, args ...any) error {
	return enrichError(ErrWrongTypeError, msg, args...)
}

var ErrNameNotFound = errors.New("name not found")

func ErrFieldNotFound(f string, fields interface{}) error {
	return enrichError(ErrNameNotFound, "field «%s» is not found in %v", f, fields)
}

var ErrInvalidName = errors.New("name not valid")

var ErrIDNotFound = errors.New("ID not found")

var ErrRecordIDNotFound = fmt.Errorf("recordID cannot be found: %w", ErrIDNotFound)

var ErrRecordNotFound = errors.New("record cannot be found")

var ErrMinOccursViolation = errors.New("minimum occurs violated")

var ErrMaxOccursViolation = errors.New("maximum occurs violated")

var ErrFieldIsEmpty = errors.New("field is empty")

var ErrInvalidVerificationKind = errors.New("invalid verification kind")

var ErrCUDsMissed = errors.New("CUDs are missed")

var ErrRawRecordIDRequired = errors.New("raw record ID required")

var ErrRawRecordIDUnexpected = errors.New("unexpected raw record ID")

var ErrRecordIDUniqueViolation = errors.New("record ID duplicates")

var ErrWrongRecordID = errors.New("wrong record ID")

var ErrUnableToUpdateSystemField = errors.New("unable to update system field")

var ErrAbstractTypeError = errors.New("abstract type")

func ErrAbstractType(msg string, args ...any) error {
	return enrichError(ErrAbstractTypeError, msg, args...)
}

var ErrUnexpectedTypeKind = errors.New("unexpected type kind")

var ErrUnknownCodec = errors.New("unknown codec")

var ErrMaxGetBatchRecordCountExceeds = errors.New("the maximum count of records to batch is exceeded")

var ErrWrongFieldTypeError = errors.New("wrong field type")

func ErrWrongFieldType(msg string, args ...any) error {
	return enrichError(ErrWrongFieldTypeError, msg, args...)
}

var ErrTypeChanged = errors.New("type has been changed")

var ErrDataConstraintViolation = errors.New("data constraint violation")

var ErrNumAppWorkspacesNotSet = errors.New("NumAppWorkspaces is not set")

var ErrCorruptedData = errors.New("corrupted data")

var ErrNullNotAllowed = errors.New("null value is not allowed")

const (
	errWrongFieldValue        = "field «%v» value should be %s, but got %T"
	errFieldValueTypeConvert  = "field «%s» value type «%T» can not to be converted to «%s»"
	errFieldMustBeVerified    = "field «%s» must be verified, token expected, but value «%T» passed"
	errFieldValueTypeMismatch = "value type «%s» is not applicable for %v"
)

const errTypedFieldNotFoundWrap = "%s-type field «%s» is not found in %v: %w" // int32-type field «myField» is not found …

const errContainerNotFoundWrap = "container «%s» is not found in type «%v»: %w" // container «order_item» is not found …

const errNumberFieldWrongValueWrap = "field «%s» value %s can not to be converted to «%s»: %w"

const errCantGetFieldQNameIDWrap = "QName field «%s» can not get ID for value «%v»: %w"

const errTypeNotFoundWrap = "type «%v» not found: %w"

const errMustValidatedBeforeStore = "%v must be validated before store: %w"

const errViewNotFoundWrap = "view «%v» not found: %w"

const errFieldDataConstraintViolatedFmt = "%v data constraint «%v» violated: %w"

// ValidateError: an interface for describing errors that occurred during validation
//   - methods:
//     — Code(): returns error code, see ECode_××× constants
type ValidateError interface {
	error
	Code() int
}
