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
			return initPackage(params.Dir)
		},
	}
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	return cmd

}

func initPackage(dir string) error {
	packageName := filepath.Base(dir)
	if err := createGoMod(dir, packageName); err != nil {
		return err
	}
	if err := createPackagesGen(nil, dir, packageName); err != nil {
		return err
	}
	if err := updateDependencies(dir); err != nil {
		return err
	}
	return nil
}

func updateDependencies(dir string) error {
	return new(exec.PipedExec).Command("go", "mod", "tidy").WorkingDir(dir).Run(nil, nil)
}

func createGoMod(dir, packageName string) error {
	filePath := filepath.Join(dir, goModFileName)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", filePath)
	}

	goVersion := runtime.Version()
	versionNumber := strings.TrimSpace(strings.TrimPrefix(goVersion, "go"))

	goModContent := fmt.Sprintf(goModContentTemplate, packageName, versionNumber)
	if err := os.WriteFile(filePath, []byte(goModContent), defaultPermissions); err != nil {
		return err
	}
	return nil
}

func createPackagesGen(imports []string, dir, packageName string) error {
	filePath := filepath.Join(dir, packagesGenFileName)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", filePath)
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
