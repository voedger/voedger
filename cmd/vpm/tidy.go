package main

import (
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/compile"
)

func newTidyCmd() *cobra.Command {
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "add missing and remove unused modules",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			params, err = prepareParams(params, args)
			if err != nil {
				return err
			}
			var appDef appdef.IAppDef
			compileRes, err := compile.Compile(params.Dir)
			if err == nil {
				appDef = compileRes.AppDef
			}
			return tidy(appDef, params.Dir)
		},
	}
	cmd.Flags().StringVarP(&params.Dir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	return cmd

}

func tidy(appDef appdef.IAppDef, dir string) error {
	packageName := filepath.Base(dir)
	if err := createPackagesGen(getImports(appDef, packageName), dir, true); err != nil {
		return err
	}
	if err := updateDependencies(dir); err != nil {
		return err
	}
	return nil
}

func getImports(appDef appdef.IAppDef, packageName string) []string {
	var imports []string
	if appDef != nil {
		exceptedPaths := []string{compile.DummyAppName, appdef.SysPackagePath, packageName}
		appDef.Packages(func(localName, fullPath string) {
			if !slices.Contains(exceptedPaths, fullPath) {
				imports = append(imports, fullPath)
			}
		})
	}
	return imports
}
