/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/untillpro/voedger/cmd/edger/internal/edger"
)

func newServerCmd() *cobra.Command {
	edgerPars := edger.EdgerParams{}

	cmd := cobra.Command{
		Use:   "server",
		Short: "Runs edger server cycle",
		Run: func(*cobra.Command, []string) {
			edger.Run(edgerPars)
		},
	}

	cmd.Flags().StringVarP(&edgerPars.AchievedStateFilePath, "ctrls.AchievedStateFilePath", "s", "",
		`full file path (include directory and file name) to load and store last achieved state.
If not assigned, then "edger-state.json" in current working directory is used.`)

	cmd.Flags().DurationVarP(&edgerPars.AchieveAttemptInterval, "edger.AchieveAttemptInterval", "a", edger.DefaultAchieveAttemptInterval,
		`time interval between achieving attempts if first attempt has finished with errors.
Minimum valid value is 10ms. 
Maximum valid value is 1h.
Default value is 500ms.`)

	return &cmd
}
