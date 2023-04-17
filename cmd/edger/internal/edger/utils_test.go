/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package edger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_sleepCtx(t *testing.T) {
	require := require.New(t)

	t.Run("basic usage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		start := time.Now()
		const interval = time.Duration(10 * time.Millisecond)
		sleepCtx(ctx, interval)
		require.LessOrEqual(interval, time.Since(start))

		cancel()
	})

	t.Run("cancelable from context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		start := time.Now()
		const interval = time.Duration(1 * time.Second)
		go func() {
			sleepCtx(ctx, interval)
			require.Greater(interval, time.Since(start))
		}()
		cancel()
	})

	t.Run("immediately returns if calls from canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		cancel()

		start := time.Now()
		const interval = time.Duration(1 * time.Second)
		sleepCtx(ctx, interval)
		require.Greater(interval, time.Since(start))
	})
}
