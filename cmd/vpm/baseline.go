/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/parser"

	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/coreutils"
)

func newBaselineCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline folder",
		Short: "create baseline schemas",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			compileRes, err := compile.Compile(params.Dir)
			if err != nil {
				return err
			}
			return baseline(compileRes, params.Dir, params.TargetDir)
		},
	}
	return cmd
}

// baseline creates baseline schemas in target dir
func baseline(compileRes *compile.Result, dir, targetDir string) error {
	if err := createBaselineDir(targetDir); err != nil {
		return err
	}

	pkgDir := filepath.Join(targetDir, pkgDirName)
	if err := saveBaselineSchemas(compileRes.PkgFiles, pkgDir); err != nil {
		return err
	}

	if err := saveBaselineInfo(compileRes, dir, targetDir); err != nil {
		return err
	}
	return nil
}

// saveBaselineInfo saves baseline info into target dir
// baseline info includes baseline package url, timestamp and git commit hash
func saveBaselineInfo(compileRes *compile.Result, dir, targetDir string) error {
	var gitCommitHash string
	sb := new(strings.Builder)
	if err := new(exec.PipedExec).Command("git", "rev-parse", "HEAD").WorkingDir(dir).Run(sb, os.Stderr); err == nil {
		gitCommitHash = strings.TrimSpace(sb.String())
	}

	baselineInfoObj := baselineInfo{
		BaselinePackageURL: compileRes.ModulePath,
		Timestamp:          time.Now().In(time.FixedZone("GMT", 0)).Format(timestampFormat),
		GitCommitHash:      gitCommitHash,
	}

	content, err := json.MarshalIndent(baselineInfoObj, "", "  ")
	if err != nil {
		return err
	}

	baselineInfoFilePath := filepath.Join(targetDir, baselineInfoFileName)
	if err := os.WriteFile(baselineInfoFilePath, content, coreutils.FileMode_rw_rw_rw_); err != nil {
		return err
	}
	if logger.IsVerbose() {
		logger.Verbose("create baseline info file: %s", baselineInfoFilePath)
	}
	return nil
}

func saveBaselineSchemas(pkgFiles packageFiles, baselineDir string) error {
	for qpn, files := range pkgFiles {
		packageDir := filepath.Join(baselineDir, qpn)
		if err := os.MkdirAll(packageDir, coreutils.FileMode_rwxrwxrwx); err != nil {
			return err
		}
		for _, file := range files {
			base := filepath.Base(file)
			fileNameExtensionless := base[:len(base)-len(filepath.Ext(base))]
			if err := coreutils.CopyFile(file, packageDir, coreutils.WithNewName(fileNameExtensionless+parser.VSQLExt)); err != nil {
				return fmt.Errorf(errFmtCopyFile, file, err)
			}
			if logger.IsVerbose() {
				filePath := filepath.Join(packageDir, fileNameExtensionless+parser.VSQLExt)
				logger.Verbose("baseline file created: %s", filePath)
			}
		}
	}
	return nil
}

func createBaselineDir(dir string) error {
	exists, err := coreutils.Exists(dir)
	if err != nil {
		// notest
		return err
	}
	if exists {
		return fmt.Errorf("baseline directory already exists: %s", dir)
	}
	pkgDir := filepath.Join(dir, pkgDirName)
	return os.MkdirAll(pkgDir, coreutils.FileMode_rwxrwxrwx)
}
