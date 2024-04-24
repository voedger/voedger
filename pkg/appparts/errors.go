/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var ErrNotFound = errors.New("not found")

var (
	ErrNotAvailableEngines                                    = errors.New("no available engines")
	errNotAvailableEngines [ProcessorKind_Count]error = [ProcessorKind_Count]error{
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Command.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Query.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Actualizer.TrimString()),
	}
)

func errAppCannotBeRedeployed(name istructs.AppQName) error {
	return fmt.Errorf("application %v can not be redeployed: %w", name, errors.ErrUnsupported)
}

func errAppNotFound(name istructs.AppQName) error {
	return fmt.Errorf("application %v not found: %w", name, ErrNotFound)
}

func errPartitionNotFound(name istructs.AppQName, partID istructs.PartitionID) error {
	return fmt.Errorf("application %v partition %v not found: %w", name, partID, ErrNotFound)
}

func errUndefinedExtension(n appdef.QName) error {
	return fmt.Errorf("undefined extension %v: %w", n, ErrNotFound)
}
