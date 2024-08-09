/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import "time"

const AppPartitionBorrowRetryDelay = 50 * time.Millisecond

// NullProcessorRunner should be used in test only
var NullProcessorRunner nullProcessorRunner = nullProcessorRunner{}
