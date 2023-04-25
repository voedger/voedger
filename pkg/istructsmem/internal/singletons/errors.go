/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package singletons

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/istructs"
)

var ErrSingletonIDsExceeds = fmt.Errorf("the maximum number of singleton document identifiers (%v) has been exceeded", istructs.MaxSingletonID)

var ErrIDNotFound = errors.New("ID not found")

var ErrWrongQNameID = errors.New("wrong QName ID")

var ErrNameNotFound = errors.New("name not found")
