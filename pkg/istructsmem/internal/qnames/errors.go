/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"errors"
	"fmt"
)

var ErrQNameIDsExceeds = fmt.Errorf("the maximum number of QName identifiers (%d) has been exceeded", MaxAvailableQNameID)

var ErrIDNotFound = errors.New("ID not found")

var ErrWrongQNameID = errors.New("wrong QName ID")

var ErrNameNotFound = errors.New("name not found")
