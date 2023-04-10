/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ctool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	t.Run("version", func(t *testing.T) {
		args := []string{"firstArg", "version"}
		require.Zero(cli(args, TestVersion))
	})
}
