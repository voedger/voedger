/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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

var ErrMissedError = errors.New("missed")

func ErrMissed(msg string, args ...any) error {
	return enrichError(ErrMissedError, msg, args...)
}

var ErrInvalidError = errors.New("not valid")

func ErrInvalid(msg string, args ...any) error {
	return enrichError(ErrInvalidError, msg, args...)
}

var ErrOutOfBoundsError = errors.New("out of bounds")

func ErrOutOfBounds(msg string, args ...any) error {
	return enrichError(ErrOutOfBoundsError, msg, args...)
}

var ErrAlreadyExistsError = errors.New("already exists")

func ErrAlreadyExists(msg string, args ...any) error {
	return enrichError(ErrAlreadyExistsError, msg, args...)
}

var ErrNotFoundError = errors.New("not found")

func ErrNotFound(msg string, args ...any) error {
	return enrichError(ErrNotFoundError, msg, args...)
}

func ErrFieldNotFound(f string) error {
	return ErrNotFound("field «%v»", f)
}

func ErrTypeNotFound(t QName) error {
	return ErrNotFound("type «%v»", t)
}

func ErrRoleNotFound(r QName) error {
	return ErrNotFound("role «%v»", r)
}

var ErrConvertError = errors.New("convert error")

func ErrConvert(msg string, args ...any) error {
	return enrichError(ErrConvertError, msg, args...)
}

var ErrTooManyError = errors.New("too many")

func ErrTooMany(msg string, args ...any) error {
	return enrichError(ErrTooManyError, msg, args...)
}

var ErrIncompatibleError = errors.New("incompatible")

func ErrIncompatible(msg string, args ...any) error {
	return enrichError(ErrIncompatibleError, msg, args...)
}

var ErrUnsupportedError = errors.ErrUnsupported

func ErrUnsupported(msg string, args ...any) error {
	return enrichError(ErrUnsupportedError, msg, args...)
}
