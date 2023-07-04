/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package bytespool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {

	require := require.New(t)

	var capacity int

	t.Run("use", func(t *testing.T) {
		b := Get()

		require.Empty(b)
		require.Zero(cap(b))

		b = append(b, []byte{1, 2, 3}...)

		capacity = cap(b)
		require.Positive(capacity)

		Put(b)
	})

	t.Run("reuse", func(t *testing.T) {
		b := Get()

		require.Empty(b)
		require.Equal(cap(b), capacity)

		Put(b)
	})
}
