/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/exp/maps"

	"github.com/voedger/voedger/cmd/vpm/internal/dm"
	"github.com/voedger/voedger/pkg/parser"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func newCompileCmd() *cobra.Command {
	params := compileParams{}
	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "compile voedger application",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				params.WorkingDir = strings.TrimPrefix(strings.TrimSpace(args[0]), "--C=")
			}
			depMan, err := dm.NewGoBasedDependencyManager()
			if err != nil {
				return err
			}
			wd, err := currentWorkingDir(params)
			if err != nil {
				return err
			}
			if err := compile(depMan, wd); err != nil {
				return err
			}
			return nil
		},
	}
	compileCmd.SilenceErrors = true
	compileCmd.Flags().StringVar(&params.WorkingDir, "C", "", "Change to dir before running the command.")
	return compileCmd
}

func compile(depMan dm.IDependencyManager, dir string) error {
	if logger.IsVerbose() {
		logger.Verbose("compilation started!")
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("working dir is %s", dir))
	}
	importedStmts := make(map[string]parser.ImportStmt)

	packages, err := compileDir(depMan, dir, testQPN, importedStmts)
	if err != nil {
		return err
	}

	if !hasAppSchema(packages) {
		appPackageAst, err := compileTestAppSchema(maps.Values(importedStmts))
		if err != nil {
			return err
		}
		packages = append(packages, appPackageAst)
		addMissingUses(appPackageAst, getUseStmts(maps.Values(importedStmts)))
	}

	sysPackageAst, err := compileSys(depMan, importedStmts)
	if err != nil {
		return err
	}
	packages = append(packages, sysPackageAst...)

	if _, err := parser.BuildAppSchema(packages); err != nil {
		return err
	}
	if logger.IsVerbose() {
		logger.Verbose("compilation finished!")
	}
	return nil
}

func hasAppSchema(packages []*parser.PackageSchemaAST) bool {
	for _, p := range packages {
		for _, f := range p.Ast.Statements {
			if f.Application != nil {
				return true
			}
		}
	}
	return false
}

func compileTestAppSchema(imports []parser.ImportStmt) (*parser.PackageSchemaAST, error) {
	fileAst := &parser.FileSchemaAST{
		FileName: sysSchemaSqlFileName,
		Ast: &parser.SchemaAST{
			Imports: imports,
			Statements: []parser.RootStatement{
				{
					Application: &parser.ApplicationStmt{
						Name: testAppName,
					},
				},
			},
		},
	}
	return parser.BuildPackageSchema(testAppName, []*parser.FileSchemaAST{fileAst})
}

func getUseStmts(imports []parser.ImportStmt) []parser.UseStmt {
	uses := make([]parser.UseStmt, len(imports))
	for i, imp := range imports {
		use := parser.Ident(path.Base(imp.Name))
		if imp.Alias != nil {
			use = *imp.Alias
		}
		uses[i] = parser.UseStmt{
			Name: use,
		}
	}
	return uses
}

func addMissingUses(appPackage *parser.PackageSchemaAST, uses []parser.UseStmt) {
	for _, f := range appPackage.Ast.Statements {
		if f.Application != nil {
			for _, use := range uses {
				found := false
				for _, useInApp := range f.Application.Uses {
					if useInApp.Name == use.Name {
						found = true
						break
					}
				}
				if !found {
					f.Application.Uses = append(f.Application.Uses, use)
				}
			}
		}
	}
}

func compileSys(depMan dm.IDependencyManager, importedStmts map[string]parser.ImportStmt) ([]*parser.PackageSchemaAST, error) {
	return compileDependency(depMan, sysPackage, importedStmts)
}

// checkImportedStmts checks if qpn is already imported. If not, it adds it to importedStmts
func checkImportedStmts(qpn string, importedStmts map[string]parser.ImportStmt) (ok bool) {
	if _, exists := importedStmts[qpn]; exists {
		return
	}
	ok = true
	importStmt := parser.ImportStmt{
		Name: qpn,
	}
	if qpn == sysPackage {
		alias := parser.Ident(qpn)
		importStmt.Alias = &alias
	}
	importedStmts[qpn] = importStmt
	return
}

func compileDir(depMan dm.IDependencyManager, dir, qpn string, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, err error) {
	if ok := checkImportedStmts(qpn, importedStmts); !ok {
		return
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("compilation of the dir %s", dir))
	}
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

func compileDependencies(depMan dm.IDependencyManager, imports []parser.ImportStmt, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, err error) {
	for _, imp := range imports {
		dependentPackages, err := compileDependency(depMan, imp.Name, importedStmts)
		if err != nil {
			return nil, err
		}
		packages = append(packages, dependentPackages...)
	}
	return packages, nil
}

func compileDependency(depMan dm.IDependencyManager, depURL string, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, err error) {
	// workaround for sys package
	depURLToFind := depURL
	if depURL == sysPackage {
		depURLToFind = sysQPN
	}
	localPath, err := depMan.LocalPath(depURLToFind)
	if err != nil {
		return nil, err
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("dependency %s is located at %s", depURL, localPath))
	}
	return compileDir(depMan, localPath, depURL, importedStmts)
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
