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

	t.Run("use", func(t *testing.T) {
		b := Get()

		require.Empty(b)

		b = append(b, []byte{1, 2, 3}...)

		Put(b)
	})

	t.Run("reuse", func(t *testing.T) {
		b := Get()

		require.Empty(b)

		b = b[:3]
		require.Equal([]byte{1, 2, 3}, b)

		Put(b)
	})
}
