/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
)

func newTidyCmd(params *vpmParams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "add missing and remove unused modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkBeforeCompile(params.Dir); err != nil {
				return err
			}

			compileRes, err := compile.Compile(params.Dir)
			if err != nil {
				logger.Verbose(err)
				fmt.Println("failed to compile, will try to exec 'go mod tidy' anyway")
			}
			if compileRes == nil {
				return errors.New("failed to compile, check schemas")
			}

			return tidy(compileRes.NotFoundDeps, compileRes.AppDef, compileRes.ModulePath, params.Dir)
		},
	}
	return cmd

}

func tidy(notFoundDeps []string, appDef appdef.IAppDef, modulePath string, dir string) error {
	// get imports and not found dependencies and try to get them via 'go get'
	imports := append(getImports(appDef, modulePath), notFoundDeps...)
	if err := getDependencies(dir, imports); err != nil {
		return err
	}
	if err := createPackagesGen(imports, dir, modulePath, true); err != nil {
		return err
	}
	return execGoModTidy(dir)
}

func getImports(appDef appdef.IAppDef, packagePath string) (imports []string) {
	if appDef == nil {
		return imports
	}
	excludedPaths := []string{compile.DummyAppName, appdef.SysPackagePath, packagePath}
	for _, fullPath := range appDef.Packages {
		if !slices.Contains(excludedPaths, fullPath) {
			imports = append(imports, fullPath)
		}
	}
	return imports
}

func getDependencies(dir string, imports []string) error {
	for _, imp := range imports {
		if strings.Contains(imp, "/") {
			if err := execGoGet(dir, imp); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBeforeCompile(dir string) error {
	if err := checkGoModFileExists(dir); err != nil {
		return errGoModFileNotFound
	}
	// packages_gen.go should be created before compiling
	exists, err := checkPackageGenFileExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("packages_gen.go not found. Run 'vpm init'")
	}
	return nil
}
