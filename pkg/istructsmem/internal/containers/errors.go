/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"errors"
	"fmt"
)

var ErrContainerIDsExceeds = fmt.Errorf("the maximum number of container identifiers (%d) has been exceeded", MaxAvailableContainerID)

var ErrContainerIDNotFound = errors.New("container ID not found")

var ErrWrongContainerID = errors.New("wrong container ID")

var ErrContainerNotFound = errors.New("container not found")
