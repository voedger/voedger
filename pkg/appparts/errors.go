/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import "errors"

var ErrNameNotFound = errors.New("name not found")

const (
	errAppNotFound       = "application %v not found: %w"
	errPartitionNotFound = "partition %v not found: %w"
)

var ErrInvalidPartitionStatus = errors.New("invalid partition status")

const (
	errPartitionAlreadyActive = "partition %v already active: %w"
	errPartitionIsInactive    = "partition %v is inactive: %w"
)
