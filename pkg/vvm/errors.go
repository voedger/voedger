/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vvm

import "errors"

var (
	ErrVVMLaunchFailure             = errors.New("VVM failed to launch")
	ErrVVMLeadershipAcquisition     = errors.New("failed to acquire leadership")
	ErrLeadershipLost               = errors.New("leadership lost")
	ErrLeadershipAcquisitionTimeout = errors.New("leadership acquisition timeout")
)
