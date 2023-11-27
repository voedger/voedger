/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/dm"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func newCompileCmd() *cobra.Command {
	params := compileParams{}
	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "compile voedger application",
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := currentWorkingDir(params)
			if err != nil {
				return err
			}
			depMan, err := dm.NewGoBasedDependencyManager(wd)
			if err != nil {
				return err
			}
			if err := compileSQLFiles(depMan, wd); err != nil {
				return err
			}
			return nil
		},
	}
	compileCmd.SilenceErrors = true
	compileCmd.Flags().StringVar(&params.WorkingDir, "C", "", "Change to dir before running the command.")
	return compileCmd
}

func compileSQLFiles(depMan dm.IDependencyManager, wd string) error {
	importedStmts := make(map[string]struct{})
	packages, err := compileDir(depMan, wd, testAQN, importedStmts)
	if err != nil {
		return err
	}
	sysPackage, err := compileSys()
	if err != nil {
		return err
	}
	if _, err := parser.BuildAppSchema(append(packages, sysPackage)); err != nil {
		return err
	}
	return nil
}

func compileSys() (*parser.PackageSchemaAST, error) {
	sysContent, err := sys.SysFS.ReadFile(sysSchemaSqlFileName)
	if err != nil {
		return nil, err
	}
	fileAst, err := parser.ParseFile(sysSchemaSqlFileName, string(sysContent))
	if err != nil {
		return nil, err
	}
	return parser.BuildPackageSchema(appdef.SysPackage, []*parser.FileSchemaAST{fileAst})
}

func compileDir(depMan dm.IDependencyManager, dir, qpn string, importedStmts map[string]struct{}) (packages []*parser.PackageSchemaAST, err error) {
	packageAst, err := parser.ParsePackageDir(qpn, coreutils.NewPathReader(dir), "")
	if err != nil {
		return nil, err
	}
	importedPackages, err := compileDependencies(depMan, packageAst.Ast.Imports, importedStmts)
	if err != nil {
		return nil, err
	}
	return append([]*parser.PackageSchemaAST{packageAst}, importedPackages...), nil
}

func compileDependencies(depMan dm.IDependencyManager, imports []parser.ImportStmt, importedStmts map[string]struct{}) (packages []*parser.PackageSchemaAST, err error) {
	for _, imp := range imports {
		if _, ok := importedStmts[imp.Name]; ok {
			continue
		}
		importedStmts[imp.Name] = struct{}{}
		dependentPackages, err := compileDependency(depMan, imp.Name, importedStmts)
		if err != nil {
			return nil, err
		}
		packages = append(packages, dependentPackages...)
	}
	return packages, nil
}

func compileDependency(depMan dm.IDependencyManager, depQPN string, importedStmts map[string]struct{}) (packages []*parser.PackageSchemaAST, err error) {
	depURL, subDir, depVersion, err := depMan.ParseDepQPN(depQPN)
	if err != nil {
		return nil, err
	}
	depCachedPath, err := depMan.ValidateDependencySubDir(depURL, depVersion, subDir)
	if err != nil {
		return nil, fmt.Errorf("dependency %s not found in %s", depURL, depMan.DependencyCachePath())
	}
	return compileDir(depMan, depCachedPath, depQPN, importedStmts)
}

func currentWorkingDir(params compileParams) (string, error) {
	if params.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("unable to get current working directory. error - %w", err)
		}
		return wd, nil
	}
	if _, err := os.Stat(params.WorkingDir); os.IsNotExist(err) {
		return "", fmt.Errorf("directory %s does not exist", params.WorkingDir)
	}
	return params.WorkingDir, nil
}

type compileParams struct {
	WorkingDir string
}
