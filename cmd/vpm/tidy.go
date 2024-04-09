/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
)

func newTidyCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "add missing and remove unused modules",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(cmd, params, args)
			if err != nil {
				return err
			}
			compileRes, err := compile.Compile(params.Dir)
			if err != nil {
				logger.Error("failed to compile, will try to exec 'go mod tidy' anyway")
				return errors.Join(err, execGoModTidy(params.Dir))
			}
			return tidy(compileRes.AppDef, compileRes.ModulePath, params.Dir)
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd

}

func tidy(appDef appdef.IAppDef, packagePath string, dir string) error {
	imports := getImports(appDef, packagePath)
	if err := createPackagesGen(imports, dir, true); err != nil {
		return err
	}
	if err := getDependencies(dir, imports); err != nil {
		return err
	}
	return execGoModTidy(dir)
}

func getImports(appDef appdef.IAppDef, packagePath string) (imports []string) {
	excludedPaths := []string{compile.DummyAppName, appdef.SysPackagePath, packagePath}
	appDef.Packages(func(localName, fullPath string) {
		if !slices.Contains(excludedPaths, fullPath) {
			imports = append(imports, fullPath)
		}
	})
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
