/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package main

import "github.com/spf13/cobra"

func newClusterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cluster",
		Short: "init",
	}
}

