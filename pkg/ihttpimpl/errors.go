/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package ihttpimpl

import "errors"

var (
	ErrAppPartNoOutOfRange       = errors.New("app part number out of range")
	ErrAppIsNotDeployed          = errors.New("app is not deployed")
	ErrAppPartitionIsNotDeployed = errors.New("app partition is not deployed")
	ErrAppAlreadyDeployed        = errors.New("app is already deployed")
	ErrActiveAppPartitionsExist  = errors.New("active app partitions exist")
)
