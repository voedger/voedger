/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package iauthnzimpl

import "errors"

var (
	ErrPersonalAccessTokenOnSystemRole = errors.New("personal access token on a system role")
	ErrPersonalAccessTokenOnNullWSID   = errors.New("personal access token on null WSID")
)
