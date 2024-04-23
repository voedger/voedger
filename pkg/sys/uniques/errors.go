/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import "errors"

var ErrUniqueConstraintViolation = errors.New("unique constraint violation")

var ErrUniqueFieldUpdateDeny = errors.New("unique field update denied")

var ErrUniqueValueTooLong = errors.New("unique value is too long")

var ErrProvidedDocCanNotHaveUniques = errors.New("type of the provided doc can not have uniques")

var ErrUniqueNotExist = errors.New("unique does not exist")
