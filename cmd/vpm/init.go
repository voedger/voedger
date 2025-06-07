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
	"golang.org/x/mod/semver"

	"github.com/voedger/voedger/pkg/compile"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/exec"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/sys"
)

var (
	// global variables used to make version checking testable
	minimalRequiredGoVersionValue     = minimalRequiredGoVersion
	minimalRequiredTinyGoVersionValue = minimalRequiredTinyGoVersion
)

func newInitCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init module-path",
		Short: "initialize a new package",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return initPackage(params.Dir, params.ModulePath)
		},
	}
	return cmd

}

func initPackage(dir, modulePath string) error {
	if modulePath == "" {
		return fmt.Errorf("vpm: cannot determine module path for source directory %s", dir)
	}
	if err := createGoMod(dir, modulePath); err != nil {
		return err
	}
	if err := createPackagesGen(nil, dir, modulePath, false); err != nil {
		return err
	}
	if err := createWasmDir(dir); err != nil {
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

func checkGoModFileExists(dir string) error {
	goModFilePath := filepath.Join(dir, goModFileName)
	exists, err := coreutils.Exists(goModFilePath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s: not found", goModFilePath)
	}
	return nil
}

func createWasmDir(dir string) error {
	exists, err := coreutils.Exists(filepath.Join(dir, wasmDirName))
	if err != nil {
		return err
	}
	if !exists {
		if err := os.Mkdir(filepath.Join(dir, wasmDirName), coreutils.FileMode_rwxrwxrwx); err != nil {
			return err
		}
	}
	return nil
}

func createGoMod(dir, modulePath string) error {
	// check go version
	goVersion := runtime.Version()

	goVersionNumber := strings.TrimSpace(strings.TrimPrefix(goVersion, "go"))
	if !checkGoVersion(goVersionNumber) {
		return fmt.Errorf(unsupportedGoVersionErrFormat, goVersionNumber)
	}

	filePath := filepath.Join(dir, goModFileName)

	exists, err := coreutils.Exists(filePath)
	if err != nil {
		// notest
		return err
	}

	// if go.mod file does not exist, create it
	if !exists {
		goModContent := fmt.Sprintf(goModContentTemplate, modulePath, goVersionNumber)
		if err := os.WriteFile(filePath, []byte(goModContent), coreutils.FileMode_rw_rw_rw_); err != nil {
			return err
		}
	}

	return execGoGet(dir, compile.VoedgerPath)
}

func checkGoVersion(goVersionNumber string) bool {
	return semver.Compare("v"+goVersionNumber, "v"+minimalRequiredGoVersionValue) >= 0
}

func checkPackageGenFileExists(dir string) (bool, error) {
	packagesGenFilePath := filepath.Join(dir, packagesGenFileName)
	return coreutils.Exists(packagesGenFilePath)
}

func createPackagesGen(imports []string, dir, modulePath string, recreate bool) error {
	// pkg subfolder for packages
	packagesGenFilePath := filepath.Join(dir, packagesGenFileName)
	if !recreate {
		exists, err := checkPackageGenFileExists(dir)
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
		if imp == sys.PackagePath {
			continue
		}
		fmt.Fprintf(strBuffer, "_ %q\n", imp)
	}

	packageName := strings.ReplaceAll(filepath.Base(modulePath), "-", "_")
	packagesGenContent := fmt.Sprintf(packagesGenContentTemplate, packageName, strBuffer.String())
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

	return new(exec.PipedExec).Command("go", "get", dependencyToGet+"@main").WorkingDir(goModDir).Run(stdout, os.Stderr)
}
