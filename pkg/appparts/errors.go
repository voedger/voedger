/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/cluster"
)

var (
	ErrNotFound          = errors.New("not found")
	errAppNotFound       = "application %v not found: %w"
	errPartitionNotFound = "application %v partition %v not found: %w"
)

var (
	ErrNotAvailableEngines                                    = errors.New("no available engines")
	errNotAvailableEngines [cluster.ProcessorKind_Count]error = [cluster.ProcessorKind_Count]error{
		fmt.Errorf("%w %s", ErrNotAvailableEngines, cluster.ProcessorKind_Command.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, cluster.ProcessorKind_Query.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, cluster.ProcessorKind_Actualizer.TrimString()),
	}
)
