/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
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
			compileRes, err := compile(newParams.WorkingDir)
			if err != nil {
				return err
			}
			if err := baseline(compileRes, newParams.TargetDir); err != nil {
				return err
			}

			return nil
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd
}

// baseline creates baseline schemas in target dir
func baseline(compileRes *compileResult, targetDir string) error {
	baselineDir, err := createBaselineDir(targetDir)
	if err != nil {
		return err
	}

	pkgDir := path.Join(baselineDir, pkgDirName)
	if err := saveBaselineSchemas(compileRes.pkgFiles, pkgDir); err != nil {
		return err
	}

	if err := saveBaselineInfo(compileRes, baselineDir); err != nil {
		return err
	}
	return nil
}

func saveBaselineInfo(compileRes *compileResult, baselineDir string) error {
	var gitCommitHash string
	if stdoutRes, _, err := new(exec.PipedExec).Command("git", "rev-parse", "HEAD").RunToStrings(); err == nil {
		gitCommitHash = strings.TrimSpace(stdoutRes)
	}

	baselineInfoObj := baselineInfo{
		BaselinePackageUrl: compileRes.modulePath,
		Timestamp:          time.Now().Format(timestampFormat),
		GitCommitHash:      gitCommitHash,
	}

	content, err := json.MarshalIndent(baselineInfoObj, "", "  ")
	if err != nil {
		return err
	}

	baselineInfoFilePath := path.Join(baselineDir, baselineInfoFileName)
	if err := os.WriteFile(baselineInfoFilePath, content, defaultPermissions); err != nil {
		return err
	}
	if logger.IsVerbose() {
		logger.Verbose("create baseline info file: %s", baselineInfoFilePath)
	}
	return nil
}

func saveBaselineSchemas(pkgFiles packageFiles, baselineDir string) error {
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
	baselineDir = path.Join(dir, baselineDirName)
	pkgDir := path.Join(baselineDir, pkgDirName)
	err = os.MkdirAll(pkgDir, defaultPermissions)
	return
}
