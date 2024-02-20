/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"github.com/spf13/cobra"
)

func newGenCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "generate",
	}
	cmd.AddCommand(newGenOrmCmd())

	initGlobalFlags(cmd, &params)
	return cmd
}

func newGenOrmCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "orm [--header_file]",
		Short: "generate ORM",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			compileResult, err := compile(params.WorkingDir)
			if err != nil {
				return err
			}
			return genOrm(compileResult)
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd
}

// genOrm generates ORM from the given working directory
func genOrm(compileRes *compileResult) error {

	return nil
}
