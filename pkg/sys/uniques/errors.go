/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import "errors"

var ErrUniqueConstraintViolation = errors.New("unique constraint violation")

var ErrUniqueFieldUpdateDeny = errors.New("unique field update denied")
