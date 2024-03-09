/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/compile"
)

func newCompileCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "compile voedger application",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}

			_, err = compile.Compile(params.Dir)
			return
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd
}

func makeAbsPath(dir string) (string, error) {
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		dir = filepath.Clean(filepath.Join(wd, dir))
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("failed to open %s", dir)
	}
	return dir, nil
}

func initGlobalFlags(cmd *cobra.Command, params *vpmParams) {
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
}

func prepareParams(params vpmParams, args []string) (newParams vpmParams, err error) {
	if len(args) > 0 {
		params.TargetDir = filepath.Clean(args[0])
	}
	newParams = params
	newParams.Dir, err = makeAbsPath(params.Dir)
	if err != nil {
		return
	}
	if newParams.IgnoreFile != "" {
		newParams.IgnoreFile = filepath.Clean(newParams.IgnoreFile)
	}
	if newParams.TargetDir == "" {
		newParams.TargetDir = newParams.Dir
	}
	return
}
