/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package projectors

import "errors"

var errChunkFull = errors.New("chunk full") // internale error to indicate that chunk for PLog read is full
