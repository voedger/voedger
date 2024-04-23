/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

var ErrMissedError = errors.New("missed")

func ErrMissed(msg string, args ...any) error {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%w: %s", ErrMissedError, s)
}

var ErrInvalidError = errors.New("not valid")

func ErrInvalid(msg string, args ...any) error {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%w: %s", ErrInvalidError, s)
}

var ErrUniqueViolationError = errors.New("unique violation")

func ErrUniqueViolation(msg string, args ...any) error {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%w: %s", ErrUniqueViolationError, s)
}

var ErrNotFoundError = errors.New("not found")

func ErrNotFound(msg string, args ...any) error {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%w: %s", ErrNotFoundError, s)
}

var ErrInvalidQNameStringRepresentation = errors.New("invalid string representation of qualified name")

var ErrInvalidTypeKind = errors.New("invalid type kind")

var ErrTooManyFields = errors.New("too many fields")

var ErrIncompatibleConstraints = errors.New("incompatible constraints")

var ErrTooManyContainers = errors.New("too many containers")

var ErrTooManyUniques = errors.New("too many uniques")

var ErrInvalidDataKind = errors.New("invalid data kind")

var ErrInvalidOccurs = errors.New("invalid occurs value")

var ErrFieldsMissed = errors.New("fields missed")

var ErrUniqueOverlaps = errors.New("unique fields overlaps")

var ErrInvalidExtensionEngineKind = errors.New("extension engine kind is not valid")

var ErrWorkspaceShouldBeAbstract = errors.New("workspace should be abstract")

var ErrInvalidProjectorEventKind = errors.New("invalid projector event kind")

var ErrEmptyProjectorEvents = errors.New("empty projector events")

var ErrInvalidProjectorCronSchedule = errors.New("invalid projector cron schedule")

var ErrScheduledProjectorWithIntents = errors.New("scheduled projector shall not have intents")

var ErrInvalidPrivilegeKind = errors.New("invalid privilege kind")

var ErrPrivilegeOnMixed = errors.New("privilege objects mixed types")
