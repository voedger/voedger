/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import "errors"

var errBatchFull = errors.New("chunk full") // internale error to indicate that chunk for PLog read is full
