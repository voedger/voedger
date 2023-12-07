/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

func newBaselineCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "create baseline schemas",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newParams, err := setUpParams(params, args)
			if err != nil {
				return err
			}
			pkgFiles, err := compile(newParams.WorkingDir)
			if err != nil {
				return err
			}
			if err := baseline(pkgFiles, newParams.TargetDir); err != nil {
				return err
			}

			return nil
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd
}

// baseline creates baseline schemas in target dir
func baseline(pkgFiles packageFiles, targetDir string) error {
	baselineDir, err := createBaselineDir(targetDir)
	if err != nil {
		return err
	}

	for qpn, files := range pkgFiles {
		packageDir := path.Join(baselineDir, qpn)
		if err := os.MkdirAll(packageDir, defaultPermissions); err != nil {
			return err
		}
		for _, file := range files {
			filePath := path.Join(packageDir, filepath.Base(file))
			fileContent, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filePath, fileContent, defaultPermissions); err != nil {
				return err
			}
			if logger.IsVerbose() {
				logger.Verbose("create baseline file: %s", filePath)
			}
		}
	}
	return nil
}

func createBaselineDir(dir string) (baselineDir string, err error) {
	baselineDir = path.Join(dir, baselineDirName, pkgDirName)
	err = os.MkdirAll(baselineDir, defaultPermissions)
	return
}
