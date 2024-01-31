// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceOperator_Errors(t *testing.T) {
	require := require.New(t)
	t.Run("panic on nil IService provided", func(t *testing.T) {
		require.Panics(func() { ServiceOperator(nil) })
	})
}
