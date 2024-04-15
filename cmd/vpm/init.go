/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/mod/semver"

	"github.com/voedger/voedger/pkg/compile"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

var minimalRequiredGoVersionValue = minimalRequiredGoVersion

func newInitCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init module-path",
		Short: "initialize a new package",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return initPackage(params.Dir, params.PackagePath)
		},
	}
	return cmd

}

func initPackage(dir, packagePath string) error {
	if packagePath == "" {
		return fmt.Errorf(packagePathIsNotDeterminedErrFormat, dir)
	}
	if err := createGoMod(dir, packagePath); err != nil {
		return err
	}
	if err := createPackagesGen(nil, dir, false); err != nil {
		return err
	}
	return execGoModTidy(dir)
}

func execGoModTidy(dir string) error {
	var stdout io.Writer
	if logger.IsVerbose() {
		stdout = os.Stdout
	}
	return new(exec.PipedExec).Command("go", "mod", "tidy").WorkingDir(dir).Run(stdout, os.Stderr)
}

func createGoMod(dir, packagePath string) error {
	filePath := filepath.Join(dir, goModFileName)

	exists, err := coreutils.Exists(filePath)
	if err != nil {
		// notest
		return err
	}
	if exists {
		return fmt.Errorf("%s already exists", filePath)
	}

	goVersion := runtime.Version()
	goVersionNumber := strings.TrimSpace(strings.TrimPrefix(goVersion, "go"))
	if !checkGoVersion(goVersionNumber) {
		return fmt.Errorf(unsupportedGoVersionErrFormat, goVersionNumber)
	}

	goModContent := fmt.Sprintf(goModContentTemplate, packagePath, goVersionNumber)
	if err := os.WriteFile(filePath, []byte(goModContent), coreutils.FileMode_rw_rw_rw_); err != nil {
		return err
	}
	if err := execGoGet(dir, compile.VoedgerPath); err != nil {
		return err
	}
	return nil
}

func checkGoVersion(goVersionNumber string) bool {
	return semver.Compare("v"+goVersionNumber, "v"+minimalRequiredGoVersionValue) >= 0
}

func createPackagesGen(imports []string, dir string, recreate bool) error {
	packagesGenFilePath := filepath.Join(dir, packagesGenFileName)
	if !recreate {
		exists, err := coreutils.Exists(packagesGenFilePath)
		if err != nil {
			// notest
			return err
		}
		if exists {
			return fmt.Errorf("%s already exists", packagesGenFilePath)
		}
	}

	strBuffer := &strings.Builder{}
	for _, imp := range imports {
		strBuffer.WriteString(fmt.Sprintf("_ %q\n", imp))
	}

	packagesGenContent := fmt.Sprintf(packagesGenContentTemplate, strBuffer.String())
	packagesGenContentFormatted, err := format.Source([]byte(packagesGenContent))
	if err != nil {
		return err
	}

	if err := os.WriteFile(packagesGenFilePath, packagesGenContentFormatted, coreutils.FileMode_rw_rw_rw_); err != nil {
		return err
	}
	return nil
}

func execGoGet(goModDir, dependencyToGet string) error {
	var stdout io.Writer
	if logger.IsVerbose() {
		stdout = os.Stdout
	}
	return new(exec.PipedExec).Command("go", "get", fmt.Sprintf("%s@main", dependencyToGet)).WorkingDir(goModDir).Run(stdout, os.Stderr)
}
