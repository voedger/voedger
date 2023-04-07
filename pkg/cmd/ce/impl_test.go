/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package ce

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)

	t.Run("version", func(t *testing.T) {
		args := []string{"firstArg", "version"}
		require.Zero(cli(args, TestVersion))
	})

	t.Run("wrong args", func(t *testing.T) {
		cases := [][]string{
			{},
			{"firstArg"},
			{"firstArg", "unknown"},
			{"firstArg", "version", "--unknown"},
			{"firstArg", "server", "--unknown"},
		}
		for _, args := range cases {
			require.Equal(1, cli(args, TestVersion))
		}
	})
}

func TestServerStartStop(t *testing.T) {
	args := []string{"firstArg", "server"}
	res := make(chan int)
	go func() {
		res <- cli(args, TestVersion)
	}()
	signals <- os.Interrupt
	require.Zero(t, <-res)
}
