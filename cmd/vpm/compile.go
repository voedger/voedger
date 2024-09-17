/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/coreutils"
)

func newCompileCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "compile voedger application",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = compile.Compile(params.Dir)
			return
		},
	}
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
	exists, err := coreutils.Exists(dir)
	if err != nil {
		// notest
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("%s not exists", dir)
	}
	return dir, nil
}

func prepareParams(cmd *cobra.Command, params *vpmParams, args []string) (err error) {
	if len(args) > 0 {
		switch {
		case strings.Contains(cmd.Use, "init"):
			params.ModulePath = args[0]
		case strings.Contains(cmd.Use, "baseline") || strings.Contains(cmd.Use, "compat"):
			params.TargetDir = filepath.Clean(args[0])
		}
	}
	params.Dir, err = makeAbsPath(params.Dir)
	if err != nil {
		return
	}
	if params.IgnoreFile != "" {
		params.IgnoreFile = filepath.Clean(params.IgnoreFile)
	}
	if params.TargetDir == "" {
		params.TargetDir = params.Dir
	}
	return nil
}
