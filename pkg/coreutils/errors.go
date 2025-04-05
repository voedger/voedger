/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package coreutils

import (
	"errors"
	"fmt"
)

var (
	ErrFieldsMissed         = errors.New("fields are missed")
	ErrFieldTypeMismatch    = errors.New("field type mismatch")
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	ErrNumberOverflow       = errors.New("number overflow")
)

func errFailedToCast(value any, to string, err error) error {
	return fmt.Errorf("failed to cast %v to %s: %w", value, to, err)
}

func errNumberOverflow(value any, to string) error {
	return fmt.Errorf("%w: %v to %s", ErrNumberOverflow, value, to)
}
