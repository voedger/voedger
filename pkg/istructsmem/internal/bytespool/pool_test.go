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

		// there is no guarantee that the garbage collector will not clean up some element of the pool in parallel with your code
		// see [issue 358](https://github.com/voedger/voedger/issues/358)
		if c := cap(b); c > 0 {
			require.Equal(c, capacity)
		}

		Put(b)
	})
}
