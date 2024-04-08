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

	"github.com/voedger/voedger/pkg/compile"
)

func newInitCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "initialize a new package",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			return initPackage(params.Dir, params.PackagePath)
		},
	}
	initGlobalFlags(cmd, &params)
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

	exists, err := exists(filePath)
	if err != nil {
		// notest
		return err
	}
	if exists {
		return fmt.Errorf("%s already exists", filePath)
	}

	goVersion := runtime.Version()
	goVersionNumber := strings.TrimSpace(strings.TrimPrefix(goVersion, "go"))
	goModContent := fmt.Sprintf(goModContentTemplate, packagePath, goVersionNumber)
	if err := os.WriteFile(filePath, []byte(goModContent), defaultPermissions); err != nil {
		return err
	}
	if err := execGoGet(dir, compile.VoedgerPath); err != nil {
		return err
	}
	return nil
}

func createPackagesGen(imports []string, dir string, recreate bool) error {
	packagesGenFilePath := filepath.Join(dir, packagesGenFileName)
	if !recreate {
		exists, err := exists(packagesGenFilePath)
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

	if err := os.WriteFile(packagesGenFilePath, packagesGenContentFormatted, defaultPermissions); err != nil {
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

func exists(filePath string) (exists bool, err error) {
	_, err = os.Stat(filePath)
	if err == nil || os.IsExist(err) {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// notest
	return false, err
}
