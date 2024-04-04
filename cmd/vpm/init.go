/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/exec"

	"github.com/voedger/voedger/pkg/compile"
)

func newInitCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new package",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return fmt.Errorf(packagePathIsNotDeterminedErrFormat, params.Dir)
			}
			return initPackage(params.Dir, args[0])
		},
	}
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
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
	if err := updateDependencies(dir); err != nil {
		return err
	}
	return nil
}

func updateDependencies(dir string) error {
	// TODO: go mod tidy's output must be logged to the user as well if error occurs
	goModFilePath := filepath.Join(dir, goModFileName)
	if _, err := os.Stat(goModFilePath); !os.IsNotExist(err) {
		return new(exec.PipedExec).Command("go", "mod", "tidy").WorkingDir(dir).Run(nil, nil)
	}
	return nil
}

func createGoMod(dir, packagePath string) error {
	filePath := filepath.Join(dir, goModFileName)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", filePath)
	}

	goVersion := runtime.Version()
	versionNumber := strings.TrimSpace(strings.TrimPrefix(goVersion, "go"))
	goModContent := fmt.Sprintf(goModContentTemplate, packagePath, versionNumber)
	if err := os.WriteFile(filePath, []byte(goModContent), defaultPermissions); err != nil {
		return err
	}
	if err := getDependency(dir, compile.VoedgerPath); err != nil {
		return err
	}
	return nil
}

func createPackagesGen(imports []string, dir string, recreate bool) error {
	filePath := filepath.Join(dir, packagesGenFileName)
	if !recreate {
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			return fmt.Errorf("%s already exists", filePath)
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

	if err := os.WriteFile(filePath, packagesGenContentFormatted, defaultPermissions); err != nil {
		return err
	}
	return nil
}

func getDependency(dir, packagePath string) error {
	return new(exec.PipedExec).Command("go", "get", fmt.Sprintf("%s@main", packagePath)).WorkingDir(dir).Run(nil, nil)
}

func getDependencies(dir string, imports []string) error {
	for _, imp := range imports {
		if strings.Contains(imp, "/") {
			if err := getDependency(dir, imp); err != nil {
				return err
			}
		}
	}
	return nil
}
