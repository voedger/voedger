/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/compile"
)

func newBuildCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "build [-C] [-o <archive-name>]",
		Short: "build",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			compileRes, err := compile.Compile(params.Dir)
			if err != nil {
				return err
			}
			return build(compileRes, params)
		},
	}
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	cmd.Flags().StringVarP(&params.Output, "output", "o", "", "output archive name")
	return cmd
}

func build(compileRes *compile.Result, params vpmParams) error {
	return nil
}
