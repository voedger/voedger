/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

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
