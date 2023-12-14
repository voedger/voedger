/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	params := vpmParams{}
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "compile voedger application",
		RunE: func(cmd *cobra.Command, args []string) error {
			newParams, err := setUpParams(params, args)
			if err != nil {
				return err
			}
			if _, err := compile(newParams.WorkingDir); err != nil {
				return err
			}
			return nil
		},
	}
	initGlobalFlags(cmd, &params)
	return cmd
}

// compile compiles schemas in working dir and returns compile result
func compile(workingDir string) (*compileResult, error) {
	depMan, err := dm.NewGoBasedDependencyManager(workingDir)
	if err != nil {
		return nil, err
	}
	var errs []error

	importedStmts := make(map[string]parser.ImportStmt)
	pkgFiles := make(packageFiles)
	goModFileDir := filepath.Dir(depMan.DependencyFilePath())
	relativeWorkingDir, err := filepath.Rel(goModFileDir, workingDir)
	if err != nil {
		return nil, err
	}
	modulePath, err := url.JoinPath(depMan.ModulePath(), relativeWorkingDir)
	if err != nil {
		return nil, err
	}
	// modulePath := filepath.Join(depMan.ModuleName(), strings.TrimPrefix(workingDir, goModFileDir))

	packages, compileDirErrs := compileDir(depMan, workingDir, modulePath, importedStmts, pkgFiles)
	errs = append(errs, compileDirErrs...)

	if !hasAppSchema(packages) {
		appPackageAst, err := getDummyAppPackageAst(maps.Values(importedStmts))
		if err != nil {
			errs = append(errs, err)
		}
		packages = append(packages, appPackageAst)
		addMissingUses(appPackageAst, getUseStmts(maps.Values(importedStmts)))
	}

	sysPackageAst, compileSysErrs := compileSys(depMan, importedStmts, pkgFiles)
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
	return &compileResult{
		modulePath:   modulePath,
		pkgFiles:     pkgFiles,
		appSchemaAST: appAst,
	}, errors.Join(errs...)
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

func getDummyAppPackageAst(imports []parser.ImportStmt) (*parser.PackageSchemaAST, error) {
	fileAst := &parser.FileSchemaAST{
		FileName: sysSchemaSqlFileName,
		Ast: &parser.SchemaAST{
			Imports: imports,
			Statements: []parser.RootStatement{
				{
					Application: &parser.ApplicationStmt{
						Name: dummyAppName,
					},
				},
			},
		},
	}
	return parser.BuildPackageSchema(dummyAppName, []*parser.FileSchemaAST{fileAst})
}

func getUseStmts(imports []parser.ImportStmt) []parser.UseStmt {
	uses := make([]parser.UseStmt, len(imports))
	for i, imp := range imports {
		use := parser.Ident(filepath.Base(imp.Name))
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

func compileSys(depMan dm.IDependencyManager, importedStmts map[string]parser.ImportStmt, pkgFiles packageFiles) ([]*parser.PackageSchemaAST, []error) {
	return compileDependency(depMan, appdef.SysPackage, importedStmts, pkgFiles)
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
	if qpn == appdef.SysPackage {
		alias := parser.Ident(qpn)
		importStmt.Alias = &alias
	}
	importedStmts[qpn] = importStmt
	return
}

func compileDir(depMan dm.IDependencyManager, dir, qpn string, importedStmts map[string]parser.ImportStmt, pkgFiles packageFiles) (packages []*parser.PackageSchemaAST, errs []error) {
	if ok := checkImportedStmts(qpn, importedStmts); !ok {
		return
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("compiling %s", dir))
	}

	packageAst, fileNames, err := parser.ParsePackageDirCollectingFiles(qpn, coreutils.NewPathReader(dir), "")
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}
	// collect all the files that belong to the package
	for _, f := range fileNames {
		pkgFiles[qpn] = append(pkgFiles[qpn], filepath.Join(dir, f))
	}
	// iterate over all imports and compile them as well
	var compileDepErrs []error
	var importedPackages []*parser.PackageSchemaAST
	if packageAst != nil {
		importedPackages, compileDepErrs = compileDependencies(depMan, packageAst.Ast.Imports, importedStmts, pkgFiles)
		errs = append(errs, compileDepErrs...)
	}
	packages = append([]*parser.PackageSchemaAST{packageAst}, importedPackages...)
	return
}

func compileDependencies(depMan dm.IDependencyManager, imports []parser.ImportStmt, importedStmts map[string]parser.ImportStmt, pkgFiles packageFiles) (packages []*parser.PackageSchemaAST, errs []error) {
	for _, imp := range imports {
		dependentPackages, compileDepErrs := compileDependency(depMan, imp.Name, importedStmts, pkgFiles)
		errs = append(errs, compileDepErrs...)
		packages = append(packages, dependentPackages...)
	}
	return
}

func compileDependency(depMan dm.IDependencyManager, depURL string, importedStmts map[string]parser.ImportStmt, pkgFiles packageFiles) (packages []*parser.PackageSchemaAST, errs []error) {
	// workaround for sys package
	depURLToFind := depURL
	if depURL == appdef.SysPackage {
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
	packages, compileDirErrs = compileDir(depMan, localPath, depURL, importedStmts, pkgFiles)
	errs = append(errs, compileDirErrs...)
	return
}

// checkWorkingDir checks if working dir is valid and returns it
func checkWorkingDir(params vpmParams) (string, error) {
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

func initGlobalFlags(cmd *cobra.Command, params *vpmParams) {
	cmd.SilenceErrors = true
	cmd.Flags().StringVarP(&params.WorkingDir, "change-dir", "C", "", "Change to dir before running the command. Any files named on the command line are interpreted after changing directories. If used, this flag must be the first one in the command line.")
}

// setUpParams sets up command line params
// returns params and error
func setUpParams(params vpmParams, args []string) (newParams vpmParams, err error) {
	newParams.WorkingDir, err = checkWorkingDir(params)
	if err != nil {
		return
	}
	if len(args) > 0 {
		for _, arg := range args {
			arg = filepath.Clean(strings.TrimSpace(arg))
			if checkFlags(&newParams.IgnoreFile, arg, []string{
				ignoreFlagWithSpace,
				ignoreFlagWithEqual,
			}) {
				continue
			}
			if checkFlags(&newParams.WorkingDir, arg, []string{
				changeDirShortFlagWithSpace,
				changeDirShortFlagWithEqual,
				changeDirFlagWithSpace,
				changeDirFlagWithEqual,
			}) {
				continue
			}
			newParams.TargetDir = arg
		}
	}
	if newParams.TargetDir == "" {
		newParams.TargetDir = newParams.WorkingDir
	}
	return
}

func checkFlags(param *string, arg string, flags []string) (ok bool) {
	for _, flag := range flags {
		if strings.HasPrefix(arg, flag) {
			*param = strings.TrimPrefix(arg, flag)
			ok = true
			return
		}
	}
	return
}
