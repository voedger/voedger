/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

var ErrNameMissed = errors.New("name is missed")

var ErrInvalidName = errors.New("name not valid")

var ErrNameUniqueViolation = errors.New("duplicate name")

var ErrNameNotFound = errors.New("name not found")

var ErrTypeNotFound = fmt.Errorf("type not found: %w", ErrNameNotFound)

var ErrFieldNotFound = fmt.Errorf("field not found: %w", ErrNameNotFound)

var ErrRoleNotFound = fmt.Errorf("role not found: %w", ErrTypeNotFound)

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
