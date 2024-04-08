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
			// notest
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		dir = filepath.Clean(filepath.Join(wd, dir))
	}
	exists, err := exists(dir)
	if err != nil {
		// notest
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("%s not exists", dir)
	}
	return dir, nil
}

func prepareParams(params vpmParams, args []string) (newParams vpmParams, err error) {
	if len(args) > 0 {
		params.TargetDir = filepath.Clean(args[0])
		params.PackagePath = args[0]
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
