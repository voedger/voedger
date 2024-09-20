/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package actualizers

import "errors"

var errBatchFull = errors.New("batch full") // internal error to indicate that the batch for reading is full

var errNoBorrowedPartition = errors.New("unexpected call to borrowedAppStructs(): no borrowed partition")
