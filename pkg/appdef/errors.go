/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 */

package appdef

import (
	"errors"
)

var ErrNameMissed = errors.New("name is missed")

var ErrInvalidName = errors.New("name not valid")

var ErrNameUniqueViolation = errors.New("duplicate name")

var ErrNameNotFound = errors.New("name not found")

var ErrInvalidQNameStringRepresentation = errors.New("invalid string representation of qualified name")

var ErrInvalidTypeKind = errors.New("invalid type kind")

var ErrTooManyFields = errors.New("too many fields")

var ErrMaxFieldLengthExceeds = errors.New("too large field length")

var ErrIncompatibleConstraints = errors.New("incompatible constraints")

var ErrTooManyContainers = errors.New("too many containers")

var ErrTooManyUniques = errors.New("too many uniques")

var ErrInvalidDataKind = errors.New("invalid data kind")

var ErrInvalidOccurs = errors.New("invalid occurs value")

var ErrFieldsMissed = errors.New("fields missed")

var ErrUniqueOverlaps = errors.New("unique fields overlaps")

var ErrExtensionEngineKindMissed = errors.New("extension engine kind is missed")

var ErrWorkspaceShouldBeAbstract = errors.New("workspace should be abstract")
