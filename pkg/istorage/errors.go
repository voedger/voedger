/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package istorage

import "errors"

var (
	ErrStorageAlreadyExists = errors.New("storage already exists")
	ErrStorageDoesNotExist  = errors.New("storage does not exist")
	ErrNoSafeAppName        = errors.New("no safe app name")
)
