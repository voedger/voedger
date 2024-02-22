/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import "errors"

var ErrNotFound = errors.New("not found")

const (
	ErrAppNotFound       = "application %v not found: %w"
	errPartitionNotFound = "application %v partition %v not found: %w"
)

var ErrNotAvailableEngines = errors.New("no available engines")
