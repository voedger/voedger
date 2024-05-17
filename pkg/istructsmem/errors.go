/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package istructsmem

import (
	"errors"
	"fmt"
)

var ErrorEventNotValid = errors.New("event is not valid")

var ErrNameMissed = errors.New("name is empty")

var ErrNameNotFound = errors.New("name not found")

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

var ErrWrongType = errors.New("wrong type")

var ErrAbstractType = errors.New("abstract type")

var ErrUnexpectedTypeKind = errors.New("unexpected type kind")

var ErrUnknownCodec = errors.New("unknown codec")

var ErrMaxGetBatchRecordCountExceeds = errors.New("the maximum count of records to batch is exceeded")

var ErrWrongFieldType = errors.New("wrong field type")

var ErrTypeChanged = errors.New("type has been changed")

var ErrDataConstraintViolation = errors.New("data constraint violation")

var ErrNumAppWorkspacesNotSet = errors.New("NumAppWorkspaces is not set")

var ErrCorruptedData = errors.New("corrupted data")

const errTypedFieldNotFoundWrap = "%s-type field «%s» is not found in %v: %w" // int32-type field «myField» is not found …

const errFieldNotFoundWrap = "field «%s» is not found in %v: %w" // int32-type field «myField» is not found …

const errContainerNotFoundWrap = "container «%s» is not found in type «%v»: %w" // container «order_item» is not found …

const errFieldValueTypeMismatchWrap = "value type «%s» is not applicable for %v: %w" // value type «float64» is not applicable for int32-field «myField»: …

const errFieldMustBeVerified = "field «%s» must be verified, token expected, but value «%T» passed: %w"

const errFieldConvertErrorWrap = "field «%s» value type «%T» can not to be converted to «%s»: %w"

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
