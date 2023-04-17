/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package edger

import (
	"context"
	"time"
)

// sleepCtx sleeps until specified interval expired or specified context `ctx` is downed
func sleepCtx(ctx context.Context, interval time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}

	select {
	case <-time.After(interval):
		return true
	case <-ctx.Done():
		return false
	}
}
