/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/exp/maps"

	"github.com/voedger/voedger/cmd/vpm/internal/dm"
	"github.com/voedger/voedger/pkg/appdef"
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
				params.WorkingDir = strings.TrimPrefix(strings.TrimSpace(args[0]), "-C ")
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
	compileCmd.Flags().StringVarP(&params.WorkingDir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
	return compileCmd
}

func compile(depMan dm.IDependencyManager, dir string) error {
	var errs []error
	importedStmts := make(map[string]parser.ImportStmt)

	packages, compileDirErrs := compileDir(depMan, dir, testQPN, importedStmts)
	errs = append(errs, compileDirErrs...)

	if !hasAppSchema(packages) {
		appPackageAst, err := compileTestAppSchema(maps.Values(importedStmts))
		if err != nil {
			errs = append(errs, err)
		}
		packages = append(packages, appPackageAst)
		addMissingUses(appPackageAst, getUseStmts(maps.Values(importedStmts)))
	}

	sysPackageAst, compileSysErrs := compileSys(depMan, importedStmts)
	packages = append(packages, sysPackageAst...)
	errs = append(errs, compileSysErrs...)

	nonNilPackages := make([]*parser.PackageSchemaAST, 0, len(packages))
	for _, p := range packages {
		if p != nil {
			nonNilPackages = append(nonNilPackages, p)
		}
	}
	appAst, err := parser.BuildAppSchema(nonNilPackages)
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}
	if appAst != nil {
		if err := parser.BuildAppDefs(appAst, appdef.New()); err != nil {
			errs = append(errs, coreutils.SplitErrors(err)...)
		}
	}
	if len(errs) == 0 {
		if logger.IsVerbose() {
			logger.Verbose("compiling succeeded")
		}
	}
	return errors.Join(errs...)
}

func hasAppSchema(packages []*parser.PackageSchemaAST) bool {
	for _, p := range packages {
		if p != nil {
			for _, f := range p.Ast.Statements {
				if f.Application != nil {
					return true
				}
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

func compileSys(depMan dm.IDependencyManager, importedStmts map[string]parser.ImportStmt) ([]*parser.PackageSchemaAST, []error) {
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

func compileDir(depMan dm.IDependencyManager, dir, qpn string, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, errs []error) {
	if ok := checkImportedStmts(qpn, importedStmts); !ok {
		return
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("compiling %s", dir))
	}
	packageAst, err := parser.ParsePackageDir(qpn, coreutils.NewPathReader(dir), "")
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}
	var compileDepErrs []error
	var importedPackages []*parser.PackageSchemaAST
	if packageAst != nil {
		importedPackages, compileDepErrs = compileDependencies(depMan, packageAst.Ast.Imports, importedStmts)
		errs = append(errs, compileDepErrs...)
	}
	packages = append([]*parser.PackageSchemaAST{packageAst}, importedPackages...)
	return
}

func compileDependencies(depMan dm.IDependencyManager, imports []parser.ImportStmt, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, errs []error) {
	for _, imp := range imports {
		dependentPackages, compileDepErrs := compileDependency(depMan, imp.Name, importedStmts)
		errs = append(errs, compileDepErrs...)
		packages = append(packages, dependentPackages...)
	}
	return
}

func compileDependency(depMan dm.IDependencyManager, depURL string, importedStmts map[string]parser.ImportStmt) (packages []*parser.PackageSchemaAST, errs []error) {
	// workaround for sys package
	depURLToFind := depURL
	if depURL == sysPackage {
		depURLToFind = sysQPN
	}
	localPath, err := depMan.LocalPath(depURLToFind)
	if err != nil {
		errs = append(errs, err)
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("dependency: %s\nlocation: %s\n", depURL, localPath))
	}
	var compileDirErrs []error
	packages, compileDirErrs = compileDir(depMan, localPath, depURL, importedStmts)
	errs = append(errs, compileDirErrs...)
	return
}

func currentWorkingDir(params compileParams) (string, error) {
	if params.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %v", err)
		}
		return wd, nil
	}
	if _, err := os.Stat(params.WorkingDir); os.IsNotExist(err) {
		return "", fmt.Errorf("failed to open %s", params.WorkingDir)
	}
	return params.WorkingDir, nil
}

type compileParams struct {
	WorkingDir string
}
